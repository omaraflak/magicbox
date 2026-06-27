package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path

func handleListAutoSendFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	folders, err := getAllAutoSendFolders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query auto-send folders: "+err.Error())
		return
	}

	for i := range folders {
		folders[i].Targets = getAutoSendTargets(folders[i].ID)
	}

	writeJSON(w, http.StatusOK, folders)
}

func getAutoSendTargets(folderID string) []AutoSendTarget {
	targets, err := getAutoSendTargetsByFolder(folderID)
	if err != nil || targets == nil {
		return []AutoSendTarget{}
	}
	return targets
}

func handleAutoSend(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if _, ok := r.URL.Query()["path"]; !ok {
			writeError(w, http.StatusBadRequest, "missing path parameter")
			return
		}
		path := r.URL.Query().Get("path")

		id, createdAt, err := getAutoSendFolderByPath(path)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]interface{}{"is_auto_send": false})
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}

		targets := getAutoSendTargets(id)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"is_auto_send": true,
			"id":           id,
			"path":         path,
			"targets":      targets,
			"created_at":   createdAt,
		})

	case http.MethodPost:
		var body struct {
			Path       string   `json:"path"`
			ContactIDs []string `json:"contact_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Prevent nested auto-send: Check if any parent path has auto-send enabled
		current := body.Path
		for {
			if current == "" {
				break
			}
			if idx := strings.LastIndex(current, "/"); idx != -1 {
				current = current[:idx]
			} else {
				current = ""
			}
			count, _ := countAutoSendFolder(current)
			if count > 0 {
				writeError(w, http.StatusBadRequest, "cannot enable auto-send: a parent folder is already configured for auto-send")
				return
			}
		}

		// Prevent nested auto-send: Check if any subfolder has auto-send enabled
		prefix := body.Path + "/%"
		subcount, _ := countAutoSendFoldersByPrefix(prefix)
		if subcount > 0 {
			writeError(w, http.StatusBadRequest, "cannot enable auto-send: a subfolder is already configured for auto-send")
			return
		}

		// First resolve contact names from the core
		client, conn, ctx, err := getCoreClient()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to contact core OS: "+err.Error())
			return
		}
		defer conn.Close()

		contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch contacts list: "+err.Error())
			return
		}

		contactMap := make(map[string]string)
		for _, c := range contactsResp.Contacts {
			contactMap[c.Id] = c.DisplayName
		}

		tx, err := beginTx()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start transaction: "+err.Error())
			return
		}
		defer tx.Rollback()

		var folderID string
		err = tx.QueryRow("SELECT id FROM auto_send_folders WHERE path = ?", body.Path).Scan(&folderID)
		if err == sql.ErrNoRows {
			folderID = uuid.NewString()
			err = insertAutoSendFolderTx(tx, folderID, body.Path, time.Now().Format(time.RFC3339))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to register folder: "+err.Error())
				return
			}
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}

		// Delete existing targets
		err = deleteAutoSendTargetsTx(tx, folderID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reset targets: "+err.Error())
			return
		}

		// Insert new targets
		for _, cid := range body.ContactIDs {
			cname, ok := contactMap[cid]
			if !ok {
				cname = cid // Fallback to ID
			}
			err = insertAutoSendTargetTx(tx, folderID, cid, cname)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to insert target: "+err.Error())
				return
			}
		}

		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to commit: "+err.Error())
			return
		}

		go sendExistingFiles(body.Path, body.ContactIDs)

		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "folder_id": folderID})

	case http.MethodDelete:
		if _, ok := r.URL.Query()["path"]; !ok {
			writeError(w, http.StatusBadRequest, "missing path parameter")
			return
		}
		path := r.URL.Query().Get("path")

		var folderID string
		folderID, err := getAutoSendFolderIDByPath(path)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]string{"message": "already disabled"})
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}

		tx, err := beginTx()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start transaction: "+err.Error())
			return
		}
		defer tx.Rollback()

		_ = deleteAutoSendTargetsTx(tx, folderID)
		_ = deleteAutoSendFolderTx(tx, folderID)

		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to commit: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "auto-send disabled"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func checkAndTriggerAutoSend(volumeName, subPath, filename string) {
	if volumeName != "storage" {
		return
	}

	current := subPath
	for {
		var folderID string
		folderID, err := getAutoSendFolderIDByPath(current)
		if err == nil {
			targets := getAutoSendTargets(folderID)
			if len(targets) > 0 {
				log.Printf("checkAndTriggerAutoSend: found Auto-Send folder at %q for file %q. Syncing to %d contacts...", current, filename, len(targets))
				go triggerAutoSendToContacts(current, subPath, filename, targets)
			}
			return
		}

		if current == "" {
			break
		}
		if idx := strings.LastIndex(current, "/"); idx != -1 {
			current = current[:idx]
		} else {
			current = ""
		}
	}
}

func triggerAutoSendToContacts(matchedFolder, subPath, filename string, targets []AutoSendTarget) {
	dirPath, err := resolvePath("storage", subPath)
	if err != nil {
		log.Printf("triggerAutoSendToContacts error: resolvePath failed: %v", err)
		return
	}
	fullPath := filepath.Join(dirPath, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("triggerAutoSendToContacts error: read file failed: %v", err)
		return
	}

	parentDir := filepath.Dir(matchedFolder)
	if matchedFolder == "" || matchedFolder == "." {
		parentDir = ""
	}

	relDestDir := subPath
	if parentDir != "" && parentDir != "." {
		rel, err := filepath.Rel(parentDir, subPath)
		if err == nil {
			relDestDir = rel
		}
	}

	transferMsg := TransferMessage{
		Filename: filename,
		Content:  content,
		DestPath: relDestDir,
	}
	payload, err := json.Marshal(transferMsg)
	if err != nil {
		return
	}

	type task struct {
		transferID string
		target     AutoSendTarget
	}
	var tasks []task

	for _, t := range targets {
		transferID := uuid.NewString()
		dbErr := insertSentHistory(transferID, filename, relDestDir, t.ContactID, t.ContactName, "sending", time.Now().Format(time.RFC3339))
		if dbErr == nil {
			tasks = append(tasks, task{transferID: transferID, target: t})
		} else {
			log.Printf("Warning: failed to record auto-send transfer state: %v", dbErr)
		}
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		log.Printf("triggerAutoSendToContacts error: failed to get core gRPC client: %v", err)
		for _, tk := range tasks {
			updateSentHistoryStatus(tk.transferID, "failed")
		}
		return
	}
	defer conn.Close()

	for _, tk := range tasks {
		log.Printf("triggerAutoSendToContacts: delivering %q to contact %s (%s)", filename, tk.target.ContactName, tk.target.ContactID)
		sendResp, err := sendWithRetry(ctx, client, &pb.SendToContactRequest{
			ContactId: tk.target.ContactID,
			AppId:     appID,
			Payload:   payload,
		})

		newStatus := "completed"
		if err != nil {
			log.Printf("triggerAutoSendToContacts: delivery to %s failed: %v", tk.target.ContactName, err)
			newStatus = "failed"
		} else if !sendResp.Success {
			log.Printf("triggerAutoSendToContacts: delivery to %s rejected: %s", tk.target.ContactName, sendResp.StatusMessage)
			newStatus = "failed"
		}

		updateSentHistoryStatusAndDate(tk.transferID, newStatus, time.Now().Format(time.RFC3339))
	}
}

func sendExistingFiles(folderPath string, contactIDs []string) {
	dirPath, err := resolvePath("storage", folderPath)
	if err != nil {
		log.Printf("sendExistingFiles error: resolvePath failed: %v", err)
		return
	}

	targets := make([]AutoSendTarget, 0, len(contactIDs))
	for _, cid := range contactIDs {
		name, _ := getAutoSendTargetName(cid)
		if name == "" {
			name = cid
		}
		targets = append(targets, AutoSendTarget{ContactID: cid, ContactName: name})
	}

	if len(targets) == 0 {
		return
	}

	type task struct {
		srcSubDir  string
		destSubDir string
		filename   string
		transferID string
		target     AutoSendTarget
	}
	var tasks []task

	parentDir := filepath.Dir(dirPath)
	if dirPath == volumes["storage"] {
		parentDir = volumes["storage"]
	}

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			relSrc, errSrc := filepath.Rel(volumes["storage"], path)
			relDest, errDest := filepath.Rel(parentDir, path)

			if errSrc == nil && errDest == nil {
				srcFolder := filepath.Dir(relSrc)
				if srcFolder == "." {
					srcFolder = ""
				}
				destFolder := filepath.Dir(relDest)
				if destFolder == "." {
					destFolder = ""
				}

				for _, t := range targets {
					transferID := uuid.NewString()
					dbErr := insertSentHistory(transferID, info.Name(), destFolder, t.ContactID, t.ContactName, "sending", time.Now().Format(time.RFC3339))
					if dbErr == nil {
						tasks = append(tasks, task{
							srcSubDir:  srcFolder,
							destSubDir: destFolder,
							filename:   info.Name(),
							transferID: transferID,
							target:     t,
						})
					}
				}
			}
		}
		return nil
	})

	go func(todo []task) {
		client, conn, ctx, err := getCoreClient()
		if err != nil {
			log.Printf("sendExistingFiles background: failed to get core client: %v", err)
			for _, tk := range todo {
				updateSentHistoryStatus(tk.transferID, "failed")
			}
			return
		}
		defer conn.Close()

		for _, tk := range todo {
			dir, err := resolvePath("storage", tk.srcSubDir)
			if err != nil {
				updateSentHistoryStatus(tk.transferID, "failed")
				continue
			}
			content, err := os.ReadFile(filepath.Join(dir, tk.filename))
			if err != nil {
				updateSentHistoryStatus(tk.transferID, "failed")
				continue
			}

			transferMsg := TransferMessage{
				Filename: tk.filename,
				Content:  content,
				DestPath: tk.destSubDir,
			}
			payload, err := json.Marshal(transferMsg)
			if err != nil {
				updateSentHistoryStatus(tk.transferID, "failed")
				continue
			}

			log.Printf("sendExistingFiles background: delivering %q to contact %s", tk.filename, tk.target.ContactName)
			sendResp, err := sendWithRetry(ctx, client, &pb.SendToContactRequest{
				ContactId: tk.target.ContactID,
				AppId:     appID,
				Payload:   payload,
			})

			newStatus := "completed"
			if err != nil {
				log.Printf("sendExistingFiles background delivery to %s failed: %v", tk.target.ContactName, err)
				newStatus = "failed"
			} else if !sendResp.Success {
				log.Printf("sendExistingFiles background delivery to %s rejected: %s", tk.target.ContactName, sendResp.StatusMessage)
				newStatus = "failed"
			}

			updateSentHistoryStatusAndDate(tk.transferID, newStatus, time.Now().Format(time.RFC3339))
		}
	}(tasks)
}
