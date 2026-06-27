package main

import (
	"net/http"
)

func handlePresets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		presets, err := getPresets()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get presets: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, presets)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name   string `json:"name"`
			Params string `json:"params"`
		}
		if !decodeBody(w, r, &req) {
			return
		}

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "Name is required")
			return
		}
		if req.Params == "" {
			req.Params = `{"temperature": 0.7, "top_k": 40, "top_p": 0.95}`
		}

		p, err := createPreset(req.Name, req.Params)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create preset: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, p)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func handlePresetDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing ID")
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			Name   string `json:"name"`
			Params string `json:"params"`
		}
		if !decodeBody(w, r, &req) {
			return
		}

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "Name is required")
			return
		}

		if err := updatePreset(id, req.Name, req.Params); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update preset: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if r.Method == http.MethodDelete {
		if err := deletePreset(id); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete preset: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}
