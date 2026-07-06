package main

import (
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/magicbox/core/sdk"
)

// --- Main ---

func main() {
	log.Println("Starting Magic Drive API on port 9090...")

	var err error
	env, err = sdk.LoadEnv()
	if err != nil {
		log.Printf("Warning: Failed to load env variables: %v", err)
	} else {
		coreURL = env.CoreURL
		apiToken = env.ApiToken
		userID = env.UserID
		appID = env.AppID
	}

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
		sdk.NewHTMLHandler("/web").ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(":9090", scopeMiddleware(mux)))
}

func scopeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if env == nil {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/contacts") || strings.HasPrefix(r.URL.Path, "/api/files/send") {
			if err := env.EnsureScopes([]string{"profile:read", "contacts:read"}, []string{"Read your user profile", "Access contacts to select file transfer recipients"}); err != nil {
				writeError(w, http.StatusForbidden, err.Error())
				return
			}
		} else if strings.HasPrefix(r.URL.Path, "/api/") {
			if err := env.EnsureScopes([]string{"profile:read", "shared:storage:rw"}, []string{"Read your user profile", "Access files in your shared storage folder"}); err != nil {
				writeError(w, http.StatusForbidden, err.Error())
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

