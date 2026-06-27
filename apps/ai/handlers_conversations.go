package main

import (
	"net/http"
	"strconv"
)

func handleConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")
		
		limit := 20
		offset := 0
		
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				limit = l
			}
		}
		if offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil {
				offset = o
			}
		}

		convs, err := getConversationsPaginated(limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get conversations: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, convs)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Title  string `json:"title"`
			Params string `json:"params"`
		}
		if !decodeBody(w, r, &req) {
			return
		}

		if req.Title == "" {
			req.Title = "New Chat"
		}
		if req.Params == "" {
			req.Params = `{"temperature": 0.7, "top_k": 40, "top_p": 0.95}`
		}

		conv, err := createConversation(req.Title, req.Params)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create conversation: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, conv)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func handleConversationDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	if r.Method == http.MethodGet {
		conv, err := getConversation(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "Conversation not found: "+err.Error())
			return
		}
		msgs, err := getMessages(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get messages: "+err.Error())
			return
		}
		
		res := struct {
			*Conversation
			Messages []Message `json:"messages"`
		}{
			Conversation: conv,
			Messages:     msgs,
		}
		
		writeJSON(w, http.StatusOK, res)
		return
	}
	
	if r.Method == http.MethodDelete {
		if err := deleteConversation(id); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete conversation: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func handleConversationParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	var req struct {
		Params string `json:"params"`
	}
	if !decodeBody(w, r, &req) {
		return
	}

	if err := updateConversationParams(id, req.Params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update params: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleConversationTitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if !decodeBody(w, r, &req) {
		return
	}

	if err := updateConversationTitle(id, req.Title); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update title: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleConversationFork(w http.ResponseWriter, r *http.Request) {
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
		MessageID string `json:"message_id"`
	}
	if !decodeBody(w, r, &req) {
		return
	}

	if req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "message_id is required")
		return
	}

	conv, err := forkConversation(id, req.MessageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fork conversation: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, conv)
}

func handleConversationForks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	forks, err := getConversationForks(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get forks: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, forks)
}

func handleConversationTreeContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	tree, err := getConversationTreeContext(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get tree context: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tree)
}

func handleUndoLastTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	content, err := removeLastTurn(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to undo last turn: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": content,
	})
}
