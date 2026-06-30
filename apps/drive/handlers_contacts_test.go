package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListContacts_Success(t *testing.T) {
	setupMockCoreServer(t)

	req := httptest.NewRequest("GET", "/api/contacts", nil)
	w := httptest.NewRecorder()
	handleListContacts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var contacts []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	json.NewDecoder(resp.Body).Decode(&contacts)
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}
	if contacts[0].DisplayName != "Alice" || contacts[1].DisplayName != "Bob" {
		t.Errorf("Unexpected contacts list in response: %+v", contacts)
	}
}

func TestHandleListContacts_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/contacts", nil)
	w := httptest.NewRecorder()
	handleListContacts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleListContacts_NoCoreClient(t *testing.T) {
	oldCoreURL := coreURL
	oldApiToken := apiToken
	coreURL = ""
	apiToken = ""
	defer func() {
		coreURL = oldCoreURL
		apiToken = oldApiToken
	}()

	req := httptest.NewRequest("GET", "/api/contacts", nil)
	w := httptest.NewRecorder()
	handleListContacts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when environment variables are missing, got %d", resp.StatusCode)
	}
}
