package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var webRoot = "/web"

func init() {
	if _, err := os.Stat("/web"); os.IsNotExist(err) {
		webRoot = "web/dist"
	}
}

func spaHandler(w http.ResponseWriter, r *http.Request) {
	// Try to serve the requested file as a static asset.
	clean := filepath.Clean(r.URL.Path)
	filePath := filepath.Join(webRoot, clean)

	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}

	// For SPA routes, serve index.html with a <base> tag injected
	// so that relative asset paths (./assets/...) always resolve
	// through the proxy prefix, regardless of the current route depth.
	indexPath := filepath.Join(webRoot, "index.html")
	html, err := os.ReadFile(indexPath)
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	// Use the proxy-forwarded prefix to determine the base path.
	basePath := "/"
	if prefix := r.Header.Get("X-Forwarded-Prefix"); prefix != "" {
		basePath = "/" + strings.Trim(prefix, "/") + "/"
	}

	baseTag := `<base href="` + basePath + `">`
	modified := strings.Replace(string(html), "<head>", "<head>\n    "+baseTag, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(modified))
}

func main() {
	log.Println("Starting Magic AI API on port 9090...")

	initDB()
	defer dbConn.Close()

	mux := http.NewServeMux()

	// Settings
	mux.HandleFunc("/api/settings", handleSettings)

	// Conversations
	mux.HandleFunc("/api/conversations", handleConversations)

	// Conversation sub-resources
	mux.HandleFunc("/api/conversations/{id}/chat", handleChat)
	mux.HandleFunc("/api/conversations/{id}/regenerate", handleRegenerate)
	mux.HandleFunc("/api/conversations/{id}/params", handleConversationParams)
	mux.HandleFunc("/api/conversations/{id}/title", handleConversationTitle)
	mux.HandleFunc("/api/conversations/{id}/forks", handleConversationForks)
	mux.HandleFunc("/api/conversations/{id}/tree-context", handleConversationTreeContext)
	mux.HandleFunc("/api/conversations/{id}/fork", handleConversationFork)
	mux.HandleFunc("/api/conversations/{id}/undo-last-turn", handleUndoLastTurn)
	mux.HandleFunc("/api/conversations/{id}", handleConversationDetail)

	// Presets
	mux.HandleFunc("/api/presets/{id}", handlePresetDetail)
	mux.HandleFunc("/api/presets", handlePresets)

	// API 404 fallback
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "unknown API endpoint")
	})

	// SPA fallback
	mux.HandleFunc("/", spaHandler)

	log.Fatal(http.ListenAndServe(":9090", mux))
}
