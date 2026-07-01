package rest

import (
	"bytes"
	"encoding/base64"
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

	payload := map[string]string{
		"multiaddr":   "QmPeerIDHere",
		"user_id":     "alice-id",
		"enc_pub_key": "test-enc-pub-key",
	}
	payloadBytes, _ := json.Marshal(payload)
	b64Payload := base64.URLEncoding.EncodeToString(payloadBytes)

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    "magicbox://invite/" + b64Payload,
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
	_ = database.AddContact("contact-id-123", userID, "Alice", "magicbox://invite/QmPeerIDHere", "target-user-id-123", "test-enc-pub-key")

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

func TestCreateContact_MissingFieldsFails(t *testing.T) {
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
	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestCreateContact_InvalidPrefixFails(t *testing.T) {
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
	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestCreateContact_InvalidBase64PayloadFails(t *testing.T) {
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
	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestCreateContact_InvalidJSONPayloadFails(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	// base64 encode a string that is not a valid JSON structure
	b64Payload := base64.URLEncoding.EncodeToString([]byte("just plain text, not JSON"))

	contactBody := map[string]string{
		"display_name": "Alice",
		"multiaddr":    "magicbox://invite/" + b64Payload,
	}
	contactBytes, _ := json.Marshal(contactBody)
	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(contactBytes))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}
