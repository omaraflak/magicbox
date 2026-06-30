package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSettings_GET_Empty(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	handleSettings(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["has_api_key"].(bool) != false {
		t.Errorf("Expected has_api_key false")
	}
	if data["api_key"].(string) != "" {
		t.Errorf("Expected api_key empty, got '%v'", data["api_key"])
	}
}

func TestHandleSettings_POST_SaveKey(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	body := strings.NewReader(`{"api_key":"my-api-key"}`)
	req := httptest.NewRequest("POST", "/api/settings", body)
	w := httptest.NewRecorder()
	handleSettings(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var postResp map[string]string
	json.NewDecoder(resp.Body).Decode(&postResp)
	if postResp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", postResp["status"])
	}

	// Verify DB state
	apiKey := getSetting("api_key")
	if apiKey != "my-api-key" {
		t.Errorf("Expected my-api-key in database, got '%s'", apiKey)
	}
}

func TestHandleSettings_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/settings", nil)
	w := httptest.NewRecorder()
	handleSettings(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
