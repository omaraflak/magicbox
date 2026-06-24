package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// MockP2PService implements p2p.Service for testing.
type MockP2PService struct {
	hostID string
}

func (m *MockP2PService) Start(ctx context.Context) error { return nil }
func (m *MockP2PService) Stop() error                    { return nil }
func (m *MockP2PService) HostID() string                 { return m.hostID }
func (m *MockP2PService) Multiaddrs() []string           { return []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + m.hostID} }
func (m *MockP2PService) RegisterHandler(appID string, handler p2p.Handler) {}
func (m *MockP2PService) SendTo(ctx context.Context, dest string, msg *p2p.Message) error {
	return nil
}

// setupTestServer initializes a fresh test server instance.
func setupTestServer(t *testing.T) (http.Handler, *db.DB, *config.Config) {
	tempDir := t.TempDir()
	tempDB := filepath.Join(tempDir, "test.db")

	database, err := db.Open(tempDB)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Root:      tempDir,
		JWTSecret: []byte("my-test-super-secret-key-signature-123"),
	}

	p2pMock := &MockP2PService{hostID: "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H"}
	orch := core.NewOrchestrator(database, nil, cfg, logger, GenerateAppToken)

	server := NewServer(cfg, database, nil, logger, orch, p2pMock)
	return server.Handler(), database, cfg
}

func TestHealth(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected health status 200, got %d", rr.Code)
	}
}

func TestSetupFlow(t *testing.T) {
	handler, _, cfg := setupTestServer(t)

	// Case 1: Initial Setup
	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected setup status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Capture the session cookie
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set on setup auto-login")
	}

	// Verify JWT claims inside the cookie
	claims, err := ValidateSessionToken(cfg.JWTSecret, sessionCookie.Value)
	if err != nil {
		t.Fatalf("failed to validate setup session token: %v", err)
	}
	if claims.Username != "omar" {
		t.Errorf("expected Username 'omar' in cookie, got %q", claims.Username)
	}

	// Case 2: Subsequent Setup blocks
	req = httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403 when setup is run again, got %d", rr.Code)
	}
}

func TestLoginFlow(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	// Create user first via setup
	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pre-setup failed: %d", rr.Code)
	}

	// Case 1: Login with invalid credentials
	loginBody := map[string]string{
		"username": "omar",
		"password": "wrongpassword",
	}
	loginBytes, _ := json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 on wrong password, got %d", rr.Code)
	}

	// Case 2: Login with valid credentials
	loginBody["password"] = "mypassword123"
	loginBytes, _ = json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 on correct login, got %d", rr.Code)
	}
}

func TestMeEndpoint(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// 1. Unauthenticated request to /me
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 when unauthenticated, got %d", rr.Code)
	}

	// Create user in DB to satisfy database lookup
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	// 2. Authenticated request to /me
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	req = httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
		Path:  "/",
	})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["username"] != "omar" {
		t.Errorf("expected username 'omar', got %v", resp["username"])
	}
}

func TestContactsEndpoint(t *testing.T) {
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

	// Case 1: Create Contact
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

	// Case 2: List Contacts
	req = httptest.NewRequest("GET", "/api/v1/contacts", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
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

func TestLogout(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	cookies := rr.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			clearedCookie = c
			break
		}
	}
	if clearedCookie == nil || clearedCookie.MaxAge != -1 {
		t.Errorf("expected cookie to be cleared (MaxAge=-1), got %+v", clearedCookie)
	}
}
