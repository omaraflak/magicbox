package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateContact_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create user in DB to satisfy database references
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{
		Name:  SessionCookieName,
		Value: token,
		Path:  "/",
	}

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    "magicbox://invite/QmPeerIDHere?user_id=alice-id",
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestListContacts_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create user in DB to satisfy database references
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	// Create a dummy contact in the DB directly
	_ = database.AddContact("contact-id-123", userID, "Alice", "magicbox://invite/QmPeerIDHere?user_id=alice-id", "target-user-id-123")

	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{
		Name:  SessionCookieName,
		Value: token,
		Path:  "/",
	}

	req := httptest.NewRequest("GET", "/api/v1/contacts", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var contacts []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&contacts)
	if len(contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(contacts))
	}
}
