package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminUpgrade_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestAdminUpgrade_ForbiddenForNonAdmin(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create standard user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)

	// Login standard user to get session cookie
	userCookie := getSessionCookieForUser(t, handler, "user", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte("{}")))
	req.AddCookie(userCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestAdminUpgrade_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u2", "admin", string(hash), true)

	// Login admin to get session cookie
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte(`{"image":"docker.io/omaraflak/magicbox-core:latest"}`)))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["message"] != "upgrade initiated successfully (mock)" {
		t.Errorf("expected mock upgrade message, got: %s", resp["message"])
	}
}

func TestAdminGetMnemonic_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Set mnemonic directly in memory.
	cfg.Mnemonic = "test mnemonic phrase"

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("GET", "/api/v1/admin/mnemonic", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["mnemonic"] != "test mnemonic phrase" {
		t.Errorf("expected mnemonic %q, got %q", "test mnemonic phrase", resp["mnemonic"])
	}
	if resp["acknowledged"] != false {
		t.Errorf("expected acknowledged=false, got %v", resp["acknowledged"])
	}
}

func TestAdminGetMnemonic_AfterAcknowledgment(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Ensure mnemonic is empty in memory.
	cfg.Mnemonic = ""

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("GET", "/api/v1/admin/mnemonic", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["mnemonic"] != "" {
		t.Errorf("expected empty mnemonic, got %q", resp["mnemonic"])
	}
	if resp["acknowledged"] != true {
		t.Errorf("expected acknowledged=true, got %v", resp["acknowledged"])
	}
}

func TestAdminGetMnemonic_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/admin/mnemonic", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestAdminAcknowledgeMnemonic_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Set mnemonic directly in memory.
	cfg.Mnemonic = "some mnemonic words"

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/mnemonic/acknowledge", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify the mnemonic was cleared from memory.
	if cfg.Mnemonic != "" {
		t.Errorf("expected Config.Mnemonic to be cleared in memory, but got %q", cfg.Mnemonic)
	}
}

func TestAdminRecoverKeys_InvalidMnemonic(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create core directory so RecoverKeys can write to it.
	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	body := bytes.NewReader([]byte(`{"mnemonic":"not a valid mnemonic"}`))
	req := httptest.NewRequest("POST", "/api/v1/admin/recover", body)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}


