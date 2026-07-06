package main

import (
	"log"
	"net/http"
	"os"

	"github.com/magicbox/core/sdk"
)

var webRoot = "/web"
var env *sdk.Env

func init() {
	if _, err := os.Stat("/web"); os.IsNotExist(err) {
		webRoot = "web/dist"
	}
}

func requireScopes(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := env.EnsureScopes([]string{"profile:read"}, []string{"Read your basic user profile (username, user ID)"}); err != nil {
			if sdk.WriteConsentError(w, err) {
				return
			}
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		next(w, r)
	}
}

func main() {
	log.Println("Starting Magic AI API on port 9090...")

	var err error
	env, err = sdk.LoadEnv()
	if err != nil {
		log.Fatalf("failed to load environment: %v", err)
	}

	initDB()
	defer dbConn.Close()

	mux := http.NewServeMux()

	// Settings
	mux.HandleFunc("/api/settings", requireScopes(handleSettings))
	mux.HandleFunc("/api/models", requireScopes(handleModels))

	// Conversations
	mux.HandleFunc("/api/conversations", requireScopes(handleConversations))

	// Conversation sub-resources
	mux.HandleFunc("/api/conversations/{id}/chat", requireScopes(handleChat))
	mux.HandleFunc("/api/conversations/{id}/regenerate", requireScopes(handleRegenerate))
	mux.HandleFunc("/api/conversations/{id}/params", requireScopes(handleConversationParams))
	mux.HandleFunc("/api/conversations/{id}/title", requireScopes(handleConversationTitle))
	mux.HandleFunc("/api/conversations/{id}/forks", requireScopes(handleConversationForks))
	mux.HandleFunc("/api/conversations/{id}/tree-context", requireScopes(handleConversationTreeContext))
	mux.HandleFunc("/api/conversations/{id}/fork", requireScopes(handleConversationFork))
	mux.HandleFunc("/api/conversations/{id}/undo-last-turn", requireScopes(handleUndoLastTurn))
	mux.HandleFunc("/api/conversations/{id}", requireScopes(handleConversationDetail))

	// Presets
	mux.HandleFunc("/api/presets/{id}", requireScopes(handlePresetDetail))
	mux.HandleFunc("/api/presets", requireScopes(handlePresets))

	// API 404 fallback
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "unknown API endpoint")
	})

	// SPA fallback
	mux.Handle("/", sdk.NewHTMLHandler(webRoot))

	log.Fatal(http.ListenAndServe(":9090", mux))
}
