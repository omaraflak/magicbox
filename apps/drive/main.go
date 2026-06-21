package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

const maxUploadSize = 50 << 20 // 50MB

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

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	uploadedFiles := r.MultipartForm.File["files"]
	if len(uploadedFiles) == 0 {
		writeError(w, http.StatusBadRequest, "no files provided in 'files' field")
		return
	}

	uploaded := 0
	for _, fh := range uploadedFiles {
		src, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to open uploaded file: "+err.Error())
			return
		}

		safeName := filepath.Base(fh.Filename)
		destPath := filepath.Join(dirPath, safeName)

		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			src.Close()
			writeError(w, http.StatusInternalServerError, "failed to create destination file: "+err.Error())
			return
		}

		_, err = io.Copy(dst, src)
		src.Close()
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

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeName))
	http.ServeFile(w, r, fullPath)
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

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("Webhook received from app=%s user=%s: %s",
		r.Header.Get("X-Magicbox-Source-App"),
		r.Header.Get("X-Magicbox-Source-User"),
		string(body),
	)
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
	mux.HandleFunc("/api/folders", handleFolders)

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
