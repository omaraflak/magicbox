package main

import (
	"archive/zip"
	"context"
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path
var volumes = map[string]string{
	"storage": "/data/shared/storage",
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
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	IsDir      bool      `json:"is_dir"`
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

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:       entry.Name(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC(),
			IsDir:      entry.IsDir(),
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

	if err := os.RemoveAll(fullPath); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
		} else {
			writeError(w, http.StatusInternalServerError, "failed to delete file/folder: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": safeName})
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

type ShareMessage struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
}

func getCoreClient() (pb.MagicboxOSClient, *grpc.ClientConn, context.Context, error) {
	if coreURL == "" || apiToken == "" {
		return nil, nil, nil, fmt.Errorf("missing gRPC core URL or authorization API token env vars")
	}

	conn, err := grpc.Dial(coreURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func handleShareFile(w http.ResponseWriter, r *http.Request) {
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
	shareMsg := ShareMessage{
		Filename: safeName,
		Content:  content,
	}
	payload, err := json.Marshal(shareMsg)
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

	writeJSON(w, http.StatusOK, map[string]string{"shared": safeName})
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var msg ShareMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Failed to parse incoming ShareMessage: %v", err)
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
		log.Println("Incoming ShareMessage has empty filename")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Create Incoming directory if it doesn't exist under user's storage
	incomingDir := filepath.Join(volumes["storage"], "Incoming")
	if err := os.MkdirAll(incomingDir, 0755); err != nil {
		log.Printf("Failed to create Incoming directory: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create Incoming folder")
		return
	}

	safeName := filepath.Base(msg.Filename)
	destPath := filepath.Join(incomingDir, safeName)

	if err := os.WriteFile(destPath, msg.Content, 0644); err != nil {
		log.Printf("Failed to write incoming file %q: %v", safeName, err)
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	log.Printf("Successfully saved shared file %q to Incoming directory", safeName)
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

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/files", handleFiles)
	mux.HandleFunc("/api/files/download", handleDownload)
	mux.HandleFunc("/api/files/download-plan", handleDownloadPlan)
	mux.HandleFunc("/api/files/move", handleMoveFile)
	mux.HandleFunc("/api/folders", handleFolders)
	mux.HandleFunc("/api/contacts", handleListContacts)
	mux.HandleFunc("/api/files/share", handleShareFile)

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

	writeJSON(w, http.StatusOK, map[string]string{"moved": safeName})
}
