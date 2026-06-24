package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path
var volumes = map[string]string{
	"storage": "/data/shared/storage",
	"trash":   "/data/shared/storage/.trash",
}

// Environment variables injected by MagicBox
var (
	apiToken = os.Getenv("MAGICBOX_API_TOKEN")
	coreURL  = os.Getenv("MAGICBOX_CORE_URL")
	userID   = os.Getenv("MAGICBOX_USER_ID")
	appID    = os.Getenv("MAGICBOX_APP_ID")
)

const (
	maxMemoryBytes    = 32 << 20  // 32MB in-memory buffer
	maxRequestBytes   = 10 << 30  // 10GB max request body size
	maxZipVolumeSize  = 200 << 20 // 200MB max split folder zip volume size
)

var cachedUsername string
var dbConn *sql.DB

func initDB() {
	dbPath := "/data/app_state/drive.db"
	os.MkdirAll("/data/app_state", 0755)

	var err error
	dbConn, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open sqlite database: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS sent_history (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			path TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'completed',
			sent_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trash (
			id TEXT PRIMARY KEY,
			original_name TEXT NOT NULL,
			original_path TEXT NOT NULL,
			trash_name TEXT NOT NULL,
			deleted_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_folders (
			id TEXT PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_targets (
			folder_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			PRIMARY KEY (folder_id, contact_id)
		)`,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			log.Fatalf("Failed to run schema queries: %v", err)
		}
	}

	// Safe DB column migration
	_, _ = dbConn.Exec("ALTER TABLE sent_history ADD COLUMN status TEXT NOT NULL DEFAULT 'completed'")
}

func startTrashCleaner() {
	// Run once immediately on startup
	go cleanExpiredTrash()

	ticker := time.NewTicker(12 * time.Hour)
	go func() {
		for range ticker.C {
			cleanExpiredTrash()
		}
	}()
}

func cleanExpiredTrash() {
	if dbConn == nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	rows, err := dbConn.Query("SELECT trash_name FROM trash WHERE deleted_at < ?", cutoff)
	if err != nil {
		log.Printf("cleanExpiredTrash: failed to query: %v", err)
		return
	}
	defer rows.Close()

	var trashNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			trashNames = append(trashNames, name)
		}
	}

	for _, name := range trashNames {
		path := filepath.Join(volumes["trash"], name)
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			log.Printf("cleanExpiredTrash: failed to remove %s: %v", path, err)
		}
		dbConn.Exec("DELETE FROM trash WHERE trash_name = ?", name)
	}
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// resolvePath returns the clean, absolute host path safely ensuring no directory traversal.
func resolvePath(volumeName, subPath string) (string, error) {
	basePath, ok := volumes[volumeName]
	if !ok {
		return "", fmt.Errorf("invalid volume: %q", volumeName)
	}

	cleanSub := filepath.Clean(subPath)
	if cleanSub == "." || cleanSub == "/" || cleanSub == "" {
		return basePath, nil
	}

	// Block relative directory traversal and absolute escape routes.
	if strings.HasPrefix(cleanSub, "/") || strings.HasPrefix(cleanSub, "..") || strings.Contains(cleanSub, "../") {
		return "", fmt.Errorf("invalid subpath: %q", subPath)
	}

	fullPath := filepath.Join(basePath, cleanSub)
	
	if !strings.HasPrefix(fullPath, basePath) {
		return "", fmt.Errorf("access denied")
	}

	return fullPath, nil
}

// --- gRPC Client helper ---

func getUsernameFromCore() (string, error) {
	if coreURL == "" || apiToken == "" {
		return "", fmt.Errorf("missing gRPC core URL or authorization API token env vars")
	}

	conn, err := grpc.Dial(coreURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to dial core gRPC server: %w", err)
	}
	defer conn.Close()

	client := pb.NewMagicboxOSClient(conn)

	// Set credentials header
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiToken))

	resp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		return "", fmt.Errorf("gRPC GetProfile call failed: %w", err)
	}

	return resp.Username, nil
}

// --- Handlers ---

type FileInfo struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	IsDir        bool      `json:"is_dir"`
	OriginalPath string    `json:"original_path,omitempty"`
	DeletedAt    *string   `json:"deleted_at,omitempty"`
	IsAutoSend   bool      `json:"is_auto_send,omitempty"`
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	username := cachedUsername
	if username == "" {
		var err error
		username, err = getUsernameFromCore()
		if err != nil {
			log.Printf("Warning: failed to fetch username via gRPC: %v", err)
			username = "User (" + userID + ")" // Fallback
		} else {
			cachedUsername = username
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":  userID,
		"app_id":   appID,
		"username": username,
	})
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListFiles(w, r)
	case http.MethodPost:
		handleUploadFiles(w, r)
	case http.MethodDelete:
		handleDeleteFile(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		writeJSON(w, http.StatusOK, []FileInfo{})
		return
	}

	if volumeName == "trash" {
		rows, err := dbConn.Query("SELECT original_name, original_path, trash_name, deleted_at FROM trash")
		var dbTrash = make(map[string]FileInfo)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var origName, origPath, trashName, delAt string
				if err := rows.Scan(&origName, &origPath, &trashName, &delAt); err == nil {
					dbTrash[trashName] = FileInfo{
						Name:         trashName,
						DisplayName:  origName,
						OriginalPath: origPath,
						DeletedAt:    &delAt,
					}
				}
			}
		}

		files := make([]FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			item, ok := dbTrash[entry.Name()]
			if !ok {
				item = FileInfo{
					Name:        entry.Name(),
					DisplayName: entry.Name(),
				}
			}
			item.Size = info.Size()
			item.ModifiedAt = info.ModTime().UTC()
			item.IsDir = entry.IsDir()

			files = append(files, item)
		}

		writeJSON(w, http.StatusOK, files)
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ".trash" || entry.Name() == ".drive.db" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		isAutoSend := false
		if entry.IsDir() && volumeName == "storage" {
			relPath := entry.Name()
			if subPath != "" {
				relPath = subPath + "/" + entry.Name()
			}
			var count int
			_ = dbConn.QueryRow("SELECT COUNT(*) FROM auto_send_folders WHERE path = ?", relPath).Scan(&count)
			isAutoSend = (count > 0)
		}

		files = append(files, FileInfo{
			Name:        entry.Name(),
			DisplayName: entry.Name(),
			Size:        info.Size(),
			ModifiedAt:  info.ModTime().UTC(),
			IsDir:       entry.IsDir(),
			IsAutoSend:  isAutoSend,
		})
	}

	writeJSON(w, http.StatusOK, files)
}

func handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Limit request body size to maxRequestBytes (e.g. 10GB)
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to open multipart reader: "+err.Error())
		return
	}

	uploaded := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read next part: "+err.Error())
			return
		}

		// Skip parts that don't have a filename (i.e. standard text form fields)
		if part.FileName() == "" {
			part.Close()
			continue
		}

		safeName := filepath.Base(part.FileName())
		destPath := filepath.Join(dirPath, safeName)

		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			part.Close()
			writeError(w, http.StatusInternalServerError, "failed to create destination file: "+err.Error())
			return
		}

		// Pipe the multipart part stream directly into the destination file!
		_, err = io.Copy(dst, part)
		part.Close()
		dst.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
			return
		}

		checkAndTriggerAutoSend(volumeName, subPath, safeName)

		uploaded++
	}

	writeJSON(w, http.StatusOK, map[string]int{"uploaded": uploaded})
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	safeName := filepath.Base(filename)
	fullPath := filepath.Join(dirPath, safeName)

	if volumeName == "trash" {
		if err := os.RemoveAll(fullPath); err != nil {
			if os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
			} else {
				writeError(w, http.StatusInternalServerError, "failed to delete file/folder: "+err.Error())
			}
			return
		}
		dbConn.Exec("DELETE FROM trash WHERE trash_name = ?", safeName)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": safeName})
		return
	}

	// Soft delete: move to trash volume
	trashDir := volumes["trash"]
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to initialize trash folder: "+err.Error())
		return
	}

	trashName := safeName + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	destPath := filepath.Join(trashDir, trashName)

	if err := os.Rename(fullPath, destPath); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
		} else {
			writeError(w, http.StatusInternalServerError, "failed to move file/folder to trash: "+err.Error())
		}
		return
	}

	trashID := uuid.NewString()
	_, dbErr := dbConn.Exec("INSERT INTO trash (id, original_name, original_path, trash_name, deleted_at) VALUES (?, ?, ?, ?, ?)",
		trashID, safeName, subPath, trashName, time.Now().Format(time.RFC3339))
	if dbErr != nil {
		log.Printf("Warning: failed to record trash metadata: %v", dbErr)
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": safeName, "trash_name": trashName})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")
	volIndexStr := r.URL.Query().Get("vol_index")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
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
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// 1. Single File Download
	if !fi.IsDir() {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeName))
		http.ServeFile(w, r, fullPath)
		return
	}

	// 2. Folder Download (zipped on the fly)
	volIndex := 0
	if volIndexStr != "" {
		volIndex, err = strconv.Atoi(volIndexStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'vol_index' parameter")
			return
		}
	}

	plan, err := generateDownloadPlan(fullPath, safeName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate plan: "+err.Error())
		return
	}

	if volIndex < 0 || volIndex >= len(plan.Volumes) {
		writeError(w, http.StatusBadRequest, "vol_index out of bounds")
		return
	}

	targetVolume := plan.Volumes[volIndex]

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", targetVolume.Name))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, relPath := range targetVolume.Files {
		fileOnDisk := filepath.Join(fullPath, relPath)
		src, err := os.Open(fileOnDisk)
		if err != nil {
			log.Printf("Failed to open file %s for zipping: %v", fileOnDisk, err)
			continue
		}

		header := &zip.FileHeader{
			Name:   filepath.ToSlash(relPath),
			Method: zip.Deflate,
		}
		header.Modified = time.Now()

		writer, err := zw.CreateHeader(header)
		if err != nil {
			src.Close()
			log.Printf("Failed to create zip header for %s: %v", relPath, err)
			continue
		}

		_, err = io.Copy(writer, src)
		src.Close()
		if err != nil {
			log.Printf("Failed to copy file %s content to zip: %v", fileOnDisk, err)
			continue
		}
	}
}

type ZipVolume struct {
	Index int      `json:"index"`
	Name  string   `json:"name"`
	Files []string `json:"files"`
	Size  int64    `json:"size"`
}

type DownloadPlan struct {
	Volumes []ZipVolume `json:"volumes"`
}

func walkDirectory(fullPath string) ([]string, map[string]int64, error) {
	files := []string{}
	sizes := make(map[string]int64)

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(fullPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			sizes[relPath] = info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Strings(files)
	return files, sizes, nil
}

func generateDownloadPlan(fullPath string, folderName string) (DownloadPlan, error) {
	files, sizes, err := walkDirectory(fullPath)
	if err != nil {
		return DownloadPlan{}, err
	}

	plan := DownloadPlan{Volumes: []ZipVolume{}}
	if len(files) == 0 {
		return plan, nil
	}

	var currentVol ZipVolume
	currentVol.Index = 0
	currentVol.Name = fmt.Sprintf("%s.part%d.zip", folderName, currentVol.Index+1)
	currentVol.Files = []string{}
	currentVol.Size = 0

	for _, file := range files {
		fileSize := sizes[file]
		if currentVol.Size+fileSize > maxZipVolumeSize && len(currentVol.Files) > 0 {
			plan.Volumes = append(plan.Volumes, currentVol)
			currentVol.Index++
			currentVol.Name = fmt.Sprintf("%s.part%d.zip", folderName, currentVol.Index+1)
			currentVol.Files = []string{}
			currentVol.Size = 0
		}
		currentVol.Files = append(currentVol.Files, file)
		currentVol.Size += fileSize
	}

	plan.Volumes = append(plan.Volumes, currentVol)

	if len(plan.Volumes) == 1 {
		plan.Volumes[0].Name = folderName + ".zip"
	}

	return plan, nil
}

func handleDownloadPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
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
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "file not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if !fi.IsDir() {
		writeJSON(w, http.StatusOK, DownloadPlan{
			Volumes: []ZipVolume{
				{
					Index: 0,
					Name:  safeName,
					Files: []string{safeName},
					Size:  fi.Size(),
				},
			},
		})
		return
	}

	plan, err := generateDownloadPlan(fullPath, safeName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate download plan: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

func handleFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "missing 'name' property")
		return
	}

	if strings.Contains(body.Name, "/") || strings.Contains(body.Name, "\\") || body.Name == "." || body.Name == ".." {
		writeError(w, http.StatusBadRequest, "invalid folder name")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	targetPath := filepath.Join(dirPath, body.Name)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create folder: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"created": body.Name})
}

type TransferMessage struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	DestPath string `json:"dest_path,omitempty"`
}

func getCoreClient() (pb.MagicboxOSClient, *grpc.ClientConn, context.Context, error) {
	if coreURL == "" || apiToken == "" {
		return nil, nil, nil, fmt.Errorf("missing gRPC core URL or authorization API token env vars")
	}

	const maxMessageSize = 512 * 1024 * 1024 // 512 MB
	conn, err := grpc.Dial(
		coreURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to dial core gRPC server: %w", err)
	}

	client := pb.NewMagicboxOSClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiToken))

	return client, conn, ctx, nil
}

func handleListContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query contacts from core: "+err.Error())
		return
	}

	type ContactJSON struct {
		ID           string `json:"id"`
		DisplayName  string `json:"display_name"`
		Multiaddr    string `json:"multiaddr"`
		TargetUserID string `json:"target_user_id"`
	}

	var contacts []ContactJSON
	for _, c := range resp.Contacts {
		contacts = append(contacts, ContactJSON{
			ID:           c.Id,
			DisplayName:  c.DisplayName,
			Multiaddr:    c.Multiaddr,
			TargetUserID: c.TargetUserId,
		})
	}

	writeJSON(w, http.StatusOK, contacts)
}

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

	content, err := os.ReadFile(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found: "+err.Error())
		return
	}

	// Prepare payload envelope
	transferMsg := TransferMessage{
		Filename: safeName,
		Content:  content,
	}
	payload, err := json.Marshal(transferMsg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal payload: "+err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	sendResp, err := client.SendToContact(ctx, &pb.SendToContactRequest{
		ContactId: contactID,
		AppId:     appID,
		Payload:   payload,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send message via core: "+err.Error())
		return
	}

	if !sendResp.Success {
		writeError(w, http.StatusBadGateway, "failed to deliver payload: "+sendResp.StatusMessage)
		return
	}

	contactName := contactID
	contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err == nil {
		for _, c := range contactsResp.Contacts {
			if c.Id == contactID {
				contactName = c.DisplayName
				break
			}
		}
	}

	transferID := uuid.NewString()
	_, dbErr := dbConn.Exec("INSERT INTO sent_history (id, filename, path, contact_id, contact_name, status, sent_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		transferID, safeName, subPath, contactID, contactName, "completed", time.Now().Format(time.RFC3339))
	if dbErr != nil {
		log.Printf("Warning: failed to record sent metadata: %v", dbErr)
	}

	writeJSON(w, http.StatusOK, map[string]string{"sent": safeName})
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
	var destPath string

	if msg.DestPath != "" {
		// Resolve target directory under primary storage
		targetDir := filepath.Join(volumes["storage"], msg.DestPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Printf("Failed to create Auto-Send target directory: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create destination folder")
			return
		}
		destPath = filepath.Join(targetDir, safeName)
	} else {
		// Create Incoming directory if it doesn't exist under user's storage
		incomingDir := filepath.Join(volumes["storage"], "Incoming")
		if err := os.MkdirAll(incomingDir, 0755); err != nil {
			log.Printf("Failed to create Incoming directory: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create Incoming folder")
			return
		}
		destPath = filepath.Join(incomingDir, safeName)
	}

	if err := os.WriteFile(destPath, msg.Content, 0644); err != nil {
		log.Printf("Failed to write incoming file %q: %v", safeName, err)
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	log.Printf("Successfully saved sent file %q to Incoming directory", safeName)
	w.WriteHeader(http.StatusOK)
}

// --- SPA serving ---

func spaHandler(w http.ResponseWriter, r *http.Request) {
	filePath := filepath.Join("/web", filepath.Clean(r.URL.Path))

	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}

	http.ServeFile(w, r, "/web/index.html")
}

// --- Main ---

func main() {
	log.Println("Starting Magic Drive API on port 8080...")

	initDB()
	defer dbConn.Close()
	startTrashCleaner()

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/files", handleFiles)
	mux.HandleFunc("/api/files/download", handleDownload)
	mux.HandleFunc("/api/files/download-plan", handleDownloadPlan)
	mux.HandleFunc("/api/files/move", handleMoveFile)
	mux.HandleFunc("/api/folders", handleFolders)
	mux.HandleFunc("/api/contacts", handleListContacts)
	mux.HandleFunc("/api/files/send", handleSendFile)
	mux.HandleFunc("/api/trash/restore", handleRestoreTrash)
	mux.HandleFunc("/api/trash/empty", handleEmptyTrash)
	mux.HandleFunc("/api/transfers", handleListTransfers)
	mux.HandleFunc("/api/transfers/file", handleListFileTransfers)
	mux.HandleFunc("/api/auto-send", handleAutoSend)
	mux.HandleFunc("/api/auto-send/all", handleListAutoSendFolders)
	mux.HandleFunc("/api/transfers/active", handleActiveTransfers)

	// Internal webhook
	mux.HandleFunc("/internal/magicbox-webhook", handleWebhook)

	// SPA fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "unknown API endpoint")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/folders") {
			writeError(w, http.StatusNotFound, "unknown API endpoint")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/internal/") {
			writeError(w, http.StatusNotFound, "unknown internal endpoint")
			return
		}
		spaHandler(w, r)
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleMoveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")
	destPath := r.URL.Query().Get("dest_path")
	newName := r.URL.Query().Get("new_name")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	srcDir, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	destDir, err := resolvePath(volumeName, destPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	safeName := filepath.Base(filename)
	srcFullPath := filepath.Join(srcDir, safeName)

	targetName := safeName
	if newName != "" {
		targetName = filepath.Base(newName)
	}
	destFullPath := filepath.Join(destDir, targetName)

	if _, err := os.Stat(srcFullPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "source file/folder not found")
		return
	}

	fi, err := os.Stat(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "destination directory not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	} else if !fi.IsDir() {
		writeError(w, http.StatusBadRequest, "destination is not a directory")
		return
	}

	if strings.HasPrefix(destFullPath, srcFullPath+string(filepath.Separator)) || destFullPath == srcFullPath {
		writeError(w, http.StatusBadRequest, "cannot move a directory into itself")
		return
	}

	if _, err := os.Stat(destFullPath); !os.IsNotExist(err) {
		writeError(w, http.StatusConflict, "a file/folder with the same name already exists in the destination folder")
		return
	}

	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to move file/folder: "+err.Error())
		return
	}

	if volumeName == "storage" {
		isDir := false
		if fi, err := os.Stat(destFullPath); err == nil {
			isDir = fi.IsDir()
		}

		if isDir {
			filepath.Walk(destFullPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					rel, err := filepath.Rel(volumes["storage"], path)
					if err == nil {
						subDir := filepath.Dir(rel)
						if subDir == "." {
							subDir = ""
						}
						checkAndTriggerAutoSend("storage", subDir, info.Name())
					}
				}
				return nil
			})
		} else {
			checkAndTriggerAutoSend(volumeName, destPath, targetName)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"moved": safeName})
}

func handleRestoreTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filename := r.URL.Query().Get("file")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	var origName, origPath string
	err := dbConn.QueryRow("SELECT original_name, original_path FROM trash WHERE trash_name = ?", filename).Scan(&origName, &origPath)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "trash file not found in database")
		} else {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}

	destDir, err := resolvePath("storage", origPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid original path: "+err.Error())
		return
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create destination folders: "+err.Error())
		return
	}

	srcFullPath := filepath.Join(volumes["trash"], filename)
	destFullPath := filepath.Join(destDir, origName)

	if _, err := os.Stat(destFullPath); !os.IsNotExist(err) {
		writeError(w, http.StatusConflict, "file/folder already exists in restored destination")
		return
	}

	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore file: "+err.Error())
		return
	}

	dbConn.Exec("DELETE FROM trash WHERE trash_name = ?", filename)

	writeJSON(w, http.StatusOK, map[string]string{"restored": origName})
}

func handleEmptyTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	trashDir := volumes["trash"]
	entries, err := os.ReadDir(trashDir)
	if err == nil {
		for _, entry := range entries {
			fullPath := filepath.Join(trashDir, entry.Name())
			if err := os.RemoveAll(fullPath); err != nil {
				log.Printf("handleEmptyTrash: failed to remove %s: %v", fullPath, err)
			}
		}
	}

	_, dbErr := dbConn.Exec("DELETE FROM trash")
	if dbErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear database records: "+dbErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "trash emptied successfully"})
}

type TransferRecord struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Path        string `json:"path"`
	ContactID   string `json:"contact_id"`
	ContactName string `json:"contact_name"`
	SentAt      string `json:"sent_at"`
}

func handleListTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rows, err := dbConn.Query("SELECT id, filename, path, contact_id, contact_name, sent_at FROM sent_history ORDER BY sent_at DESC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query sent history: "+err.Error())
		return
	}
	defer rows.Close()

	records := []TransferRecord{}
	for rows.Next() {
		var rec TransferRecord
		if err := rows.Scan(&rec.ID, &rec.Filename, &rec.Path, &rec.ContactID, &rec.ContactName, &rec.SentAt); err == nil {
			records = append(records, rec)
		}
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

	rows, err := dbConn.Query("SELECT id, filename, path, contact_id, contact_name, sent_at FROM sent_history WHERE filename = ? AND path = ? ORDER BY sent_at DESC", filename, subPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query file sent history: "+err.Error())
		return
	}
	defer rows.Close()

	records := []TransferRecord{}
	for rows.Next() {
		var rec TransferRecord
		if err := rows.Scan(&rec.ID, &rec.Filename, &rec.Path, &rec.ContactID, &rec.ContactName, &rec.SentAt); err == nil {
			records = append(records, rec)
		}
	}

	writeJSON(w, http.StatusOK, records)
}

type AutoSendTarget struct {
	ContactID   string `json:"contact_id"`
	ContactName string `json:"contact_name"`
}

type AutoSendFolderInfo struct {
	ID        string           `json:"id"`
	Path      string           `json:"path"`
	Targets   []AutoSendTarget `json:"targets"`
	CreatedAt string           `json:"created_at"`
}

func handleListAutoSendFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rows, err := dbConn.Query("SELECT id, path, created_at FROM auto_send_folders")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query auto-send folders: "+err.Error())
		return
	}
	defer rows.Close()

	var folders []AutoSendFolderInfo
	for rows.Next() {
		var f AutoSendFolderInfo
		if err := rows.Scan(&f.ID, &f.Path, &f.CreatedAt); err == nil {
			f.Targets = getAutoSendTargets(f.ID)
			folders = append(folders, f)
		}
	}

	writeJSON(w, http.StatusOK, folders)
}

func getAutoSendTargets(folderID string) []AutoSendTarget {
	rows, err := dbConn.Query("SELECT contact_id, contact_name FROM auto_send_targets WHERE folder_id = ?", folderID)
	if err != nil {
		return []AutoSendTarget{}
	}
	defer rows.Close()

	var targets []AutoSendTarget
	for rows.Next() {
		var t AutoSendTarget
		if err := rows.Scan(&t.ContactID, &t.ContactName); err == nil {
			targets = append(targets, t)
		}
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

		var id, createdAt string
		err := dbConn.QueryRow("SELECT id, created_at FROM auto_send_folders WHERE path = ?", path).Scan(&id, &createdAt)
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

		tx, err := dbConn.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start transaction: "+err.Error())
			return
		}
		defer tx.Rollback()

		var folderID string
		err = tx.QueryRow("SELECT id FROM auto_send_folders WHERE path = ?", body.Path).Scan(&folderID)
		if err == sql.ErrNoRows {
			folderID = uuid.NewString()
			_, err = tx.Exec("INSERT INTO auto_send_folders (id, path, created_at) VALUES (?, ?, ?)",
				folderID, body.Path, time.Now().Format(time.RFC3339))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to register folder: "+err.Error())
				return
			}
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}

		// Delete existing targets
		_, err = tx.Exec("DELETE FROM auto_send_targets WHERE folder_id = ?", folderID)
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
			_, err = tx.Exec("INSERT INTO auto_send_targets (folder_id, contact_id, contact_name) VALUES (?, ?, ?)",
				folderID, cid, cname)
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
		err := dbConn.QueryRow("SELECT id FROM auto_send_folders WHERE path = ?", path).Scan(&folderID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]string{"message": "already disabled"})
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}

		tx, err := dbConn.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start transaction: "+err.Error())
			return
		}
		defer tx.Rollback()

		_, _ = tx.Exec("DELETE FROM auto_send_targets WHERE folder_id = ?", folderID)
		_, _ = tx.Exec("DELETE FROM auto_send_folders WHERE id = ?", folderID)

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
		err := dbConn.QueryRow("SELECT id FROM auto_send_folders WHERE path = ?", current).Scan(&folderID)
		if err == nil {
			targets := getAutoSendTargets(folderID)
			if len(targets) > 0 {
				log.Printf("checkAndTriggerAutoSend: found Auto-Send folder at %q for file %q. Syncing to %d contacts...", current, filename, len(targets))
				go triggerAutoSendToContacts(subPath, filename, targets)
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

func triggerAutoSendToContacts(subPath, filename string, targets []AutoSendTarget) {
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

	transferMsg := TransferMessage{
		Filename: filename,
		Content:  content,
		DestPath: subPath,
	}
	payload, err := json.Marshal(transferMsg)
	if err != nil {
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		return
	}
	defer conn.Close()

	for _, t := range targets {
		transferID := uuid.NewString()
		_, dbErr := dbConn.Exec("INSERT INTO sent_history (id, filename, path, contact_id, contact_name, status, sent_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			transferID, filename, subPath, t.ContactID, t.ContactName, "sending", time.Now().Format(time.RFC3339))
		if dbErr != nil {
			log.Printf("Warning: failed to record auto-send transfer state: %v", dbErr)
		}

		log.Printf("triggerAutoSendToContacts: delivering %q to contact %s (%s)", filename, t.ContactName, t.ContactID)
		sendResp, err := client.SendToContact(ctx, &pb.SendToContactRequest{
			ContactId: t.ContactID,
			AppId:     appID,
			Payload:   payload,
		})

		newStatus := "completed"
		if err != nil {
			log.Printf("triggerAutoSendToContacts: delivery to %s failed: %v", t.ContactName, err)
			newStatus = "failed"
		} else if !sendResp.Success {
			log.Printf("triggerAutoSendToContacts: delivery to %s rejected: %s", t.ContactName, sendResp.StatusMessage)
			newStatus = "failed"
		}

		_, _ = dbConn.Exec("UPDATE sent_history SET status = ?, sent_at = ? WHERE id = ?",
			newStatus, time.Now().Format(time.RFC3339), transferID)
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
		var name string
		_ = dbConn.QueryRow("SELECT contact_name FROM auto_send_targets WHERE contact_id = ?", cid).Scan(&name)
		if name == "" {
			name = cid
		}
		targets = append(targets, AutoSendTarget{ContactID: cid, ContactName: name})
	}

	if len(targets) == 0 {
		return
	}

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, err := filepath.Rel(volumes["storage"], path)
			if err == nil {
				subDir := filepath.Dir(rel)
				if subDir == "." {
					subDir = ""
				}
				log.Printf("sendExistingFiles: triggering sync for existing file %q under %q", info.Name(), subDir)
				triggerAutoSendToContacts(subDir, info.Name(), targets)
			}
		}
		return nil
	})
}

func handleActiveTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM sent_history WHERE status = 'sending'").Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query active transfers: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"active_count": count})
}

