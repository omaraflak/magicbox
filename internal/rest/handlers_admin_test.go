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

	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/keymanager"
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
	cfg.Keys.Mnemonic = "test mnemonic phrase"
	cfg.Keys.IdentityKeyIndex = 0
	cfg.Keys.EncryptionKeyIndex = 0

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
	if resp["identity_key_index"] == nil {
		t.Errorf("expected identity_key_index to be present")
	} else if int(resp["identity_key_index"].(float64)) != 0 {
		t.Errorf("expected identity_key_index to be 0, got %v", resp["identity_key_index"])
	}
	if resp["encryption_key_index"] == nil {
		t.Errorf("expected encryption_key_index to be present")
	} else if int(resp["encryption_key_index"].(float64)) != 0 {
		t.Errorf("expected encryption_key_index to be 0, got %v", resp["encryption_key_index"])
	}
}

func TestAdminGetMnemonic_AfterAcknowledgment(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Ensure mnemonic is empty in memory.
	cfg.Keys.Mnemonic = ""
	cfg.Keys.IdentityKeyIndex = 0
	cfg.Keys.EncryptionKeyIndex = 0

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
	if resp["identity_key_index"] == nil {
		t.Errorf("expected identity_key_index to be present")
	} else if int(resp["identity_key_index"].(float64)) != 0 {
		t.Errorf("expected identity_key_index to be 0, got %v", resp["identity_key_index"])
	}
	if resp["encryption_key_index"] == nil {
		t.Errorf("expected encryption_key_index to be present")
	} else if int(resp["encryption_key_index"].(float64)) != 0 {
		t.Errorf("expected encryption_key_index to be 0, got %v", resp["encryption_key_index"])
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
	cfg.Keys.Mnemonic = "some mnemonic words"

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
	if cfg.Keys.Mnemonic != "" {
		t.Errorf("expected Config.Keys.Mnemonic to be cleared in memory, but got %q", cfg.Keys.Mnemonic)
	}
}

func TestAdminRotateEncryptionKeys_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create core directory so keys can be written.
	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Generate a valid mnemonic
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	// Add a contact to verify the propagation loop runs without crash
	_ = database.AddContact("c1", "u1", "Friend", "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "/ip4/127.0.0.1/tcp/5001/p2p/QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "friend-user-id", "some-enc-pub-key")

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"mnemonic": mnemonic,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate-encryption", bytes.NewReader(bodyBytes))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	if cfg.Keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected EncryptionKeyIndex to be 1, got %d", cfg.Keys.EncryptionKeyIndex)
	}

	val, err := database.GetSystemSetting(db.SettingEncryptionKeyIndex)
	if err != nil {
		t.Fatalf("failed to get db setting: %v", err)
	}
	if val != "1" {
		t.Errorf("expected db setting to be '1', got %q", val)
	}
}

func TestAdminRotateEncryptionKeys_InvalidMnemonic(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"mnemonic": "invalid mnemonic words here",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate-encryption", bytes.NewReader(bodyBytes))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAdminResetIdentityKeys_SuccessGenerated(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/keys/reset-identity", bytes.NewReader([]byte("{}")))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["mnemonic"] == "" {
		t.Errorf("expected generated mnemonic in response")
	}

	if cfg.Keys.IdentityKeyIndex != 0 || cfg.Keys.EncryptionKeyIndex != 0 {
		t.Errorf("expected indices to be reset to 0 in config")
	}

	val1, _ := database.GetSystemSetting(db.SettingIdentityKeyIndex)
	val2, _ := database.GetSystemSetting(db.SettingEncryptionKeyIndex)
	if val1 != "0" || val2 != "0" {
		t.Errorf("expected indices to be reset to '0' in db")
	}
}

func TestAdminResetIdentityKeys_SuccessProvided(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	mnemonic, _ := crypto.GenerateMnemonic()
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"mnemonic": mnemonic,
	})

	req := httptest.NewRequest("POST", "/api/v1/admin/keys/reset-identity", bytes.NewReader(bodyBytes))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["mnemonic"] != mnemonic {
		t.Errorf("expected mnemonic %q, got %q", mnemonic, resp["mnemonic"])
	}
}

func TestAdminRotateIdentityKeys_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Pre-populate settings and contact
	mnemonic, _ := crypto.GenerateMnemonic()
	_ = database.SetSystemSetting(db.SettingIdentityKeyIndex, "0")
	cfg.Keys.IdentityKeyIndex = 0
	cfg.Keys.Mnemonic = mnemonic

	// We must write dummy keys first so keymanager can do things (actually RotateIdentity overwrites them anyway, but it validates the mnemonic)
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 0, 0)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"mnemonic": mnemonic,
	})

	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate-identity", bytes.NewReader(bodyBytes))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	val, _ := database.GetSystemSetting(db.SettingIdentityKeyIndex)
	if val != "1" {
		t.Errorf("expected identity key index in DB to be updated to 1, got %s", val)
	}
	if cfg.Keys.IdentityKeyIndex != 1 {
		t.Errorf("expected identity key index in config to be updated to 1, got %d", cfg.Keys.IdentityKeyIndex)
	}
}

func TestAdminRestart_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u2", "admin", string(hash), true)

	// Login admin to get session cookie
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/restart", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["message"] != "restarting" {
		t.Errorf("expected restarting, got: %s", resp["message"])
	}
}



