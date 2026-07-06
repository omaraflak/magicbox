package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleProfile_Success(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/profile", nil)
	rr := httptest.NewRecorder()

	handleProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["user_id"] != "test-user-id" || resp["username"] != "test-user" {
		t.Errorf("Unexpected profile response: %+v", resp)
	}
}

func TestHandleProfile_MethodNotAllowed(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/profile", nil)
	rr := httptest.NewRecorder()

	handleProfile(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", rr.Code)
	}
}
