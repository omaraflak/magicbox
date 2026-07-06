package main

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/magicbox/core/sdk"
)

var (
	env      *sdk.Env
	coreURL  string
	apiToken string
	userID   string
	appID    string
)

var (
	messageClients = make(map[string]chan struct{})
	clientsMu      sync.Mutex
)

func notifyClients() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for _, ch := range messageClients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func main() {
	log.Println("Starting Magic Chat API on port 9090...")

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

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/profile", handleProfile)
	mux.HandleFunc("/api/contacts", handleContacts)
	mux.HandleFunc("/api/contacts/add", handleAddContact)
	mux.HandleFunc("/api/conversations", handleConversations)
	mux.HandleFunc("/api/conversations/", handleConversationRoutes) // /api/conversations/{id}/messages etc.
	mux.HandleFunc("/api/events", handleEvents)

	// File attachments server
	mux.Handle("/api/attachments/", http.StripPrefix("/api/attachments/", http.FileServer(http.Dir("/data/shared/storage/Chat"))))

	// Webhook endpoint
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

	// Start background message delivery queue processor
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		// Run initial scan at startup after a short delay
		time.Sleep(5 * time.Second)
		processDeliveryQueue()
		for range ticker.C {
			processDeliveryQueue()
		}
	}()

	log.Fatal(http.ListenAndServe(":9090", mux))
}

// Helpers for writing responses

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Disable caching for APIs
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	
	// Encode JSON
	_ = jsonEncode(w, data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// EventSource client stream
func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering for proxy servers (Nginx/Traefik)

	ch := make(chan struct{}, 1)
	clientsMu.Lock()
	clientKey := r.RemoteAddr + "-" + time.Now().Format("150405.000000")
	messageClients[clientKey] = ch
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(messageClients, clientKey)
		clientsMu.Unlock()
		close(ch)
	}()

	// Send initial ping to establish connection
	_, _ = w.Write([]byte("data: connected\n\n"))
	w.(http.Flusher).Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			_, _ = w.Write([]byte("data: update\n\n"))
			w.(http.Flusher).Flush()
		case <-ticker.C:
			// keep alive ping
			_, _ = w.Write([]byte("data: ping\n\n"))
			w.(http.Flusher).Flush()
		}
	}
}
