package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeBody(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return false
	}
	return true
}

func streamSSE(w http.ResponseWriter, ch <-chan string, errCh <-chan error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported!")
		return
	}

	chOpen := true
	errChOpen := true

	for chOpen || errChOpen {
		select {
		case msg, ok := <-ch:
			if !ok {
				chOpen = false
			} else {
				data, _ := json.Marshal(map[string]string{"content": msg})
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()
			}
		case err, ok := <-errCh:
			if !ok {
				errChOpen = false
			} else {
				data, _ := json.Marshal(map[string]string{"error": err.Error()})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(data))
				flusher.Flush()
			}
		}
	}
}
