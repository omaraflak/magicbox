package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path

func handleSendFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")
	contactID := r.URL.Query().Get("contact_id")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}
	if contactID == "" {
		writeError(w, http.StatusBadRequest, "missing 'contact_id' parameter")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	safeName := filepath.Base(filename)
	fullPath := filepath.Join(dirPath, safeName)

	fi, err := os.Stat(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found: "+err.Error())
		return
	}

	contactName := contactID
	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	// Check if the contact has the drive app installed
	checkResp, err := client.IsAppInstalled(ctx, &pb.IsAppInstalledRequest{
		ContactId: contactID,
		AppId:     "com.magicbox.drive",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check if app is installed on contact: "+err.Error())
		return
	}
	if !checkResp.Installed {
		writeError(w, http.StatusBadRequest, "Recipient contact does not have Magic Drive installed.")
		return
	}

	contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err == nil {
		for _, c := range contactsResp.Contacts {
			if c.Id == contactID {
				contactName = c.DisplayName
				break
			}
		}
	}

	type task struct {
		srcSubDir  string
		destSubDir string
		filename   string
		transferID string
	}
	var tasks []task

	if !fi.IsDir() {
		// Single file send: destination is root of Received ("")
		transferID := uuid.NewString()
		dbErr := insertSentHistory(transferID, safeName, "", contactID, contactName, "sending", time.Now().Format(time.RFC3339))
		if dbErr != nil {
			log.Printf("Warning: failed to record sent metadata: %v", dbErr)
		}
		tasks = append(tasks, task{
			srcSubDir:  subPath,
			destSubDir: "",
			filename:   safeName,
			transferID: transferID,
		})
	} else {
		// Folder send: walk and queue all files relative to parent of the sent folder
		parentDir := filepath.Dir(fullPath)
		filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				// Calculate source subDir on disk (relative to storage volumes)
				relSrc, errSrc := filepath.Rel(volumes["storage"], path)
				// Calculate destination subDir relative to parent of the folder being sent
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

					transferID := uuid.NewString()
					dbErr := insertSentHistory(transferID, info.Name(), destFolder, contactID, contactName, "sending", time.Now().Format(time.RFC3339))
					if dbErr == nil {
						tasks = append(tasks, task{
							srcSubDir:  srcFolder,
							destSubDir: destFolder,
							filename:   info.Name(),
							transferID: transferID,
						})
					}
				}
			}
			return nil
		})
	}

	// Process tasks in background sequentially
	go func(todo []task, cID, cName string) {
		client, conn, ctx, err := getCoreClient()
		if err != nil {
			log.Printf("async multi-send: failed to get core client: %v", err)
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

			log.Printf("async multi-send: delivering %q to contact %s", tk.filename, cName)
			sendResp, err := sendWithRetry(ctx, client, &pb.SendToContactRequest{
				ContactId: cID,
				AppId:     appID,
				Payload:   payload,
			})

			newStatus := "completed"
			if err != nil {
				log.Printf("async multi-send delivery to %s failed: %v", cName, err)
				newStatus = "failed"
			} else if !sendResp.Success {
				log.Printf("async multi-send delivery to %s rejected: %s", cName, sendResp.StatusMessage)
				newStatus = "failed"
			}

			updateSentHistoryStatusAndDate(tk.transferID, newStatus, time.Now().Format(time.RFC3339))
		}
	}(tasks, contactID, contactName)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "sending",
		"filename": safeName,
	})
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var msg TransferMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Failed to parse incoming TransferMessage: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	sourceType := r.Header.Get("X-Magicbox-Source-Type")
	sourceUser := r.Header.Get("X-Magicbox-Source-User")
	log.Printf("Webhook received from app=%s user=%s type=%s filename=%s size=%d",
		r.Header.Get("X-Magicbox-Source-App"),
		sourceUser,
		sourceType,
		msg.Filename,
		len(msg.Content),
	)

	if msg.Filename == "" {
		log.Println("Incoming TransferMessage has empty filename")
		w.WriteHeader(http.StatusOK)
		return
	}

	safeName := filepath.Base(msg.Filename)

	targetDir := filepath.Join(volumes["storage"], "Received", msg.DestPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Printf("Failed to create Received target directory: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create Received folder")
		return
	}

	uniqueName := resolveUniqueFilename(targetDir, safeName)
	destPath := filepath.Join(targetDir, uniqueName)

	if err := os.WriteFile(destPath, msg.Content, 0644); err != nil {
		log.Printf("Failed to write incoming file %q: %v", uniqueName, err)
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	log.Printf("Successfully saved sent file %q to Received directory", uniqueName)
	w.WriteHeader(http.StatusOK)
}

// --- SPA serving ---

func handleListTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	records, err := getAllTransfers(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query sent history: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, records)
}

func handleListFileTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filename := r.URL.Query().Get("file")
	subPath := r.URL.Query().Get("path")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	records, err := getSentHistoryByFile(filename, subPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query file sent history: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, records)
}

func handleActiveTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	count, err := countActiveTransfers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query active transfers: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"active_count": count})
}

func handleActiveListTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	recentCompletedCutoff := time.Now().Add(-10 * time.Second).Format(time.RFC3339)
	recentFailedCutoff := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	records, err := getActiveTransfers(recentFailedCutoff, recentCompletedCutoff)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query active list: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, records)
}
