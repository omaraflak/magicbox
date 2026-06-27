package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Volume mapping: logical name → filesystem path

func spaHandler(w http.ResponseWriter, r *http.Request) {
	filePath := filepath.Join("/web", filepath.Clean(r.URL.Path))

	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}

	// Inject <base> tag so relative asset paths resolve through the proxy prefix.
	html, err := os.ReadFile("/web/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	basePath := "/"
	if prefix := r.Header.Get("X-Forwarded-Prefix"); prefix != "" {
		basePath = "/" + strings.Trim(prefix, "/") + "/"
	}

	baseTag := `<base href="` + basePath + `">`
	modified := strings.Replace(string(html), "<head>", "<head>\n    "+baseTag, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(modified))
}

// --- Main ---

func main() {
	log.Println("Starting Magic Drive API on port 9090...")

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
	mux.HandleFunc("/api/files/paste", handlePaste)
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
	mux.HandleFunc("/api/transfers/active-list", handleActiveListTransfers)

	// Internal webhook
	mux.HandleFunc("/internal/magicbox-webhook", handleWebhook)

	// SPA fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "unknown API endpoint")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/internal/") {
			writeError(w, http.StatusNotFound, "unknown internal endpoint")
			return
		}
		spaHandler(w, r)
	})

	log.Fatal(http.ListenAndServe(":9090", mux))
}
