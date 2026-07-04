package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magicbox/core/internal/invite"
)

func TestSendContactRequest_Success(t *testing.T) {
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

	payload := &invite.Payload{
		Multiaddr: "/ip4/127.0.0.1/tcp/4001/p2p/QmPeerIDHere",
		UserID:    "alice-id",
		EncPubKey: "test-enc-pub-key",
	}
	inviteLink, _ := invite.Build(payload)

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    inviteLink,
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts/request", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify outgoing request was stored.
	reqs, err := database.GetContactRequests(userID, "outgoing")
	if err != nil {
		t.Fatalf("GetContactRequests failed: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 outgoing request, got %d", len(reqs))
	}
	if reqs[0].DisplayName != "Alice" {
		t.Errorf("expected display_name Alice, got %s", reqs[0].DisplayName)
	}
}

func TestListContacts_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create user in DB to satisfy database references
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	// Create a dummy contact in the DB directly
	_ = database.AddContact("contact-id-123", userID, "Alice", "QmPeerIDHere", "/ip4/127.0.0.1/tcp/4001/p2p/QmPeerIDHere", "target-user-id-123", "test-enc-pub-key", "test-master-pub-key")

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
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var contacts []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &contacts); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if len(contacts) != 1 || contacts[0]["display_name"] != "Alice" {
		t.Errorf("unexpected contacts response: %+v", contacts)
	}
}

func TestSendContactRequest_MissingFieldsFails(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	contactBody := map[string]string{
		"display_name": "",
		"multiaddr":    "magicbox://invite/anything",
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts/request", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestSendContactRequest_InvalidPrefixFails(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    "http://some-url/invite/anything",
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts/request", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestSendContactRequest_InvalidBase64PayloadFails(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    "magicbox://invite/not-valid-base64-payload!!!",
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts/request", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestSendContactRequest_InvalidJSONPayloadFails(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	// Build a valid invite link prefix with invalid JSON content inside the base64.
	inviteLink := "magicbox://invite/" + "anVzdCBwbGFpbiB0ZXh0LCBub3QgSlNPTg=="

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    inviteLink,
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts/request", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}
