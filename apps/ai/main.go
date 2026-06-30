package main

import (
	"log"
	"net/http"
	"os"

	"github.com/magicbox/core/sdk"
)

var webRoot = "/web"

func init() {
	if _, err := os.Stat("/web"); os.IsNotExist(err) {
		webRoot = "web/dist"
	}
}

func main() {
	log.Println("Starting Magic AI API on port 9090...")

	initDB()
	defer dbConn.Close()

	mux := http.NewServeMux()

	// Settings
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/models", handleModels)

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
	mux.Handle("/", sdk.NewHTMLHandler(webRoot))

	log.Fatal(http.ListenAndServe(":9090", mux))
}
