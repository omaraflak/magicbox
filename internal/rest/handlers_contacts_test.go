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

func TestSendContactRequest_AutoAccept(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	// Insert an existing incoming contact request from remote-user-id to local userID
	err := database.InsertContactRequest(
		"req-id-123", userID, "incoming", "Bob (Incoming)",
		"QmPeerIDHere", "/ip4/127.0.0.1/tcp/4001/p2p/QmPeerIDHere", "remote-user-id", "remote-enc-pub-key", "remote-master-pub-key",
	)
	if err != nil {
		t.Fatalf("failed to insert incoming request: %v", err)
	}

	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{
		Name:  SessionCookieName,
		Value: token,
		Path:  "/",
	}

	// Generate a valid mock invite link targeting the same remote user
	payload := &invite.Payload{
		Multiaddr:    "/ip4/127.0.0.1/tcp/4001/p2p/QmPeerIDHere",
		UserID:       "remote-user-id",
		EncPubKey:    "remote-enc-pub-key",
		MasterPubKey: "remote-master-pub-key",
	}
	inviteLink, _ := invite.Build(payload)

	contactBody := map[string]string{
		"display_name": "Bob (Accepted)",
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

	// Verify incoming request was deleted.
	reqObj, err := database.GetContactRequest(userID, "req-id-123")
	if err != nil {
		t.Fatalf("GetContactRequest failed: %v", err)
	}
	if reqObj != nil {
		t.Error("expected incoming contact request to be deleted, but it still exists")
	}

	// Verify contact was created.
	c, err := database.GetContactByTargetUserID(userID, "remote-user-id")
	if err != nil {
		t.Fatalf("GetContactByTargetUserID failed: %v", err)
	}
	if c == nil {
		t.Fatal("expected contact to be created in database, got nil")
	}
	if c.DisplayName != "Bob (Incoming)" {
		t.Errorf("expected DisplayName 'Bob (Incoming)' (from the incoming request), got %q", c.DisplayName)
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

func TestAcceptContactRequest_NewContact(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	// Insert an incoming contact request
	reqID := "req-abc"
	err := database.InsertContactRequest(
		reqID, userID, "incoming", "Alice",
		"peer-alice-1", "/ip4/1.1.1.1/p2p/peer-alice-1",
		"alice-uid", "alice-enc-pub-1", "alice-master-pub-1",
	)
	if err != nil {
		t.Fatalf("failed to insert contact request: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/contacts/requests/req-abc/accept", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify contact was created
	c, err := database.GetContactByTargetUserID(userID, "alice-uid")
	if err != nil {
		t.Fatalf("failed to query contact: %v", err)
	}
	if c == nil {
		t.Fatal("expected contact to be created")
	}
	if c.DisplayName != "Alice" || c.Status != "active" {
		t.Errorf("unexpected contact state: %+v", c)
	}

	// Verify contact request was deleted
	reqs, _ := database.GetContactRequests(userID, "incoming")
	if len(reqs) != 0 {
		t.Errorf("expected 0 incoming requests, got %d", len(reqs))
	}
}

func TestAcceptContactRequest_ExistingContact(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	cookie := &http.Cookie{Name: SessionCookieName, Value: token, Path: "/"}

	// Insert existing contact with status 'revoked'
	existingID := "existing-alice"
	err := database.AddContact(
		existingID, userID, "Alice",
		"peer-alice-old", "/ip4/1.1.1.1/p2p/peer-alice-old",
		"alice-uid", "alice-enc-pub-old", "alice-master-pub-old",
	)
	if err != nil {
		t.Fatalf("failed to add contact: %v", err)
	}
	database.UpdateContactStatus(existingID, "revoked")

	// Insert incoming request with new credentials (reconnect request)
	reqID := "req-xyz"
	err = database.InsertContactRequest(
		reqID, userID, "incoming", "Alice",
		"peer-alice-new", "/ip4/2.2.2.2/p2p/peer-alice-new",
		"alice-uid", "alice-enc-pub-new", "alice-master-pub-new",
	)
	if err != nil {
		t.Fatalf("failed to insert contact request: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/contacts/requests/req-xyz/accept", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify existing contact was updated, NOT created as new
	contacts, err := database.GetContacts(userID)
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}

	c := contacts[0]
	if c.ID != existingID {
		t.Errorf("expected contact ID %q, got %q", existingID, c.ID)
	}
	if c.PeerID != "peer-alice-new" || c.Multiaddr != "/ip4/2.2.2.2/p2p/peer-alice-new" || c.EncPubKey != "alice-enc-pub-new" || c.MasterPubKey != "alice-master-pub-new" {
		t.Errorf("contact fields not updated correctly: %+v", c)
	}
	if c.Status != "active" {
		t.Errorf("expected status 'active', got %q", c.Status)
	}

	// Verify request was deleted
	reqs, _ := database.GetContactRequests(userID, "incoming")
	if len(reqs) != 0 {
		t.Errorf("expected 0 incoming requests, got %d", len(reqs))
	}
}
