package main

import (
	"net/http"
)

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		apiKey := getSetting("api_key")
		hasKey := apiKey != ""
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"has_api_key": hasKey,
			"api_key":     apiKey,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req map[string]string
		if !decodeBody(w, r, &req) {
			return
		}

		apiKey := req["api_key"]
		if apiKey != "" {
			if err := setSetting("api_key", apiKey); err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to save settings: "+err.Error())
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}
