package main

import (
	"net/http"
)

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if !decodeBody(w, r, &req) {
		return
	}

	ch := make(chan string)
	errCh := make(chan error)

	go chatStream(r.Context(), id, req.Message, ch, errCh)

	streamSSE(w, ch, errCh)
}

func handleRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	lastUserMsg, err := removeLastTurn(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove last turn: "+err.Error())
		return
	}

	ch := make(chan string)
	errCh := make(chan error)

	go chatStream(r.Context(), id, lastUserMsg, ch, errCh)

	streamSSE(w, ch, errCh)
}
