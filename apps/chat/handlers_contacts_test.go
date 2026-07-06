package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "github.com/magicbox/core/api/proto/v1"
)

func TestHandleContacts_Success(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/contacts", nil)
	rr := httptest.NewRecorder()

	handleContacts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	var contacts []*pb.Contact
	if err := json.Unmarshal(rr.Body.Bytes(), &contacts); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(contacts) != 2 || contacts[0].DisplayName != "Alice" {
		t.Errorf("Unexpected contacts response: %+v", contacts)
	}
}

func TestHandleAddContact_Success(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	body := `{"invite_link":"magicbox://invite/alice-link","display_name":"Alice"}`
	req := httptest.NewRequest("POST", "/api/contacts/add", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleAddContact(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddContact_BadRequest(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Missing display_name
	body := `{"invite_link":"magicbox://invite/alice-link"}`
	req := httptest.NewRequest("POST", "/api/contacts/add", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleAddContact(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", rr.Code)
	}
}
