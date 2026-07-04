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

func TestAdminRotateKeys_EncryptionOnly(t *testing.T) {
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

	// Pre-populate keys on disk so RotateEncryption can read the index
	err = keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}
	cfg.Keys.EncryptionKeyIndex = 1

	// Add a contact to verify the propagation loop runs without crash
	_ = database.AddContact("c1", "u1", "Friend", "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "/ip4/127.0.0.1/tcp/5001/p2p/QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "friend-user-id", "some-enc-pub-key", "friend-master-pub-key")

	cfg.MnemonicStore.Set(mnemonic)

	body, _ := json.Marshal(map[string]bool{
		"rotate_encryption": true,
		"rotate_identity":   false,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	if cfg.Keys.EncryptionKeyIndex != 2 {
		t.Errorf("expected EncryptionKeyIndex to be 2, got %d", cfg.Keys.EncryptionKeyIndex)
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	val, err := os.ReadFile(paths.EncryptionIndexPath)
	if err != nil {
		t.Fatalf("failed to read encryption index file: %v", err)
	}
	if string(val) != "2" {
		t.Errorf("expected encryption index file content to be '2', got %q", string(val))
	}
}

func TestAdminRotateKeys_Locked(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Ensure system is locked (mnemonic store is empty)
	cfg.MnemonicStore.Set("")

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	body, _ := json.Marshal(map[string]bool{
		"rotate_encryption": true,
		"rotate_identity":   false,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Errorf("expected 412 Precondition Failed, got %d", rr.Code)
	}
}


func TestAdminResetIdentityKeys_SuccessionRecovery(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Setup mnemonic and keys
	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.Keys.IdentityKeyIndex = 1
	cfg.Keys.EncryptionKeyIndex = 1
	cfg.MnemonicStore.Set(mnemonic)

	// Write keys to disk
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	// Update cfg's MasterPublicKeyPEM
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}
	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		t.Fatalf("failed to marshal master public key: %v", err)
	}
	cfg.Keys.MasterPublicKeyPEM = masterPubPEM

	// Create admin user (id: u1)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Add a contact for u1
	_ = database.AddContact("c1", "u1", "Friend", "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "/ip4/127.0.0.1/tcp/5001/p2p/QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "friend-user-id", "some-enc-pub-key", "friend-master-pub-key")

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
	expectedMsg := "Identity keys rotated and succession certificate queued for all contacts. Contacts preserved."
	if resp["message"] != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, resp["message"])
	}

	// Verify contacts are NOT wiped
	contacts, err := database.GetContacts("u1")
	if err != nil {
		t.Fatalf("failed to get contacts: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != "c1" {
		t.Errorf("expected contact c1 to be preserved, got %d contacts", len(contacts))
	}

	// Verify indices are incremented
	if cfg.Keys.IdentityKeyIndex != 2 || cfg.Keys.EncryptionKeyIndex != 2 {
		t.Errorf("expected indices to be incremented to 2 in config, got identity=%d, encryption=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	idIdx, _ := os.ReadFile(paths.IdentityIndexPath)
	encIdx, _ := os.ReadFile(paths.EncryptionIndexPath)
	if string(idIdx) != "2" || string(encIdx) != "2" {
		t.Errorf("expected indices to be incremented to '2' on disk, got identity=%s, encryption=%s", string(idIdx), string(encIdx))
	}

	// Verify succession certificate is enqueued
	msgs, err := database.GetPendingMessages()
	if err != nil {
		t.Fatalf("failed to get pending messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 pending message, got %d", len(msgs))
	}
	if msgs[0].AppID != "system:key-succession" {
		t.Errorf("expected message AppID 'system:key-succession', got %q", msgs[0].AppID)
	}
}

func TestAdminResetIdentityKeys_NuclearReset_DifferentMnemonic(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Setup mnemonic and keys
	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.Keys.IdentityKeyIndex = 1
	cfg.Keys.EncryptionKeyIndex = 1
	cfg.MnemonicStore.Set(mnemonic)

	// Write keys to disk
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	// Update cfg's MasterPublicKeyPEM
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}
	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		t.Fatalf("failed to marshal master public key: %v", err)
	}
	cfg.Keys.MasterPublicKeyPEM = masterPubPEM

	// Create admin user (id: u1)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Add a contact for u1
	_ = database.AddContact("c1", "u1", "Friend", "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "/ip4/127.0.0.1/tcp/5001/p2p/QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "friend-user-id", "some-enc-pub-key", "friend-master-pub-key")

	differentMnemonic, _ := crypto.GenerateMnemonic()
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"mnemonic": differentMnemonic,
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
	expectedMsg := "P2P identity reset successfully. Revocation and reconnect requests sent to all contacts. Restart required."
	if resp["message"] != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, resp["message"])
	}
	if resp["mnemonic"] != differentMnemonic {
		t.Errorf("expected mnemonic %q, got %q", differentMnemonic, resp["mnemonic"])
	}

	// Verify contacts are NOT wiped
	contacts, err := database.GetContacts("u1")
	if err != nil {
		t.Fatalf("failed to get contacts: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != "c1" {
		t.Errorf("expected contact c1 to be preserved, got %d contacts", len(contacts))
	}

	// Verify indices are reset to 1
	if cfg.Keys.IdentityKeyIndex != 1 || cfg.Keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected indices to be reset to 1 in config, got identity=%d, encryption=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	idIdx, _ := os.ReadFile(paths.IdentityIndexPath)
	encIdx, _ := os.ReadFile(paths.EncryptionIndexPath)
	if string(idIdx) != "1" || string(encIdx) != "1" {
		t.Errorf("expected indices to be reset to '1' on disk, got identity=%s, encryption=%s", string(idIdx), string(encIdx))
	}

	// Verify messages in queue
	msgs, err := database.GetPendingMessages()
	if err != nil {
		t.Fatalf("failed to get pending messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 pending messages, got %d", len(msgs))
	}
	appIDs := map[string]bool{msgs[0].AppID: true, msgs[1].AppID: true}
	if !appIDs["system:master-revocation"] || !appIDs["system:contact-request"] {
		t.Errorf("expected master-revocation and contact-request messages, got: %+v", msgs)
	}
}

func TestAdminResetIdentityKeys_NuclearReset_EmptyBody(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Setup mnemonic and keys
	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.Keys.IdentityKeyIndex = 1
	cfg.Keys.EncryptionKeyIndex = 1
	cfg.MnemonicStore.Set(mnemonic)

	// Write keys to disk
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	// Update cfg's MasterPublicKeyPEM
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}
	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		t.Fatalf("failed to marshal master public key: %v", err)
	}
	cfg.Keys.MasterPublicKeyPEM = masterPubPEM

	// Create admin user (id: u1)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Add a contact for u1
	_ = database.AddContact("c1", "u1", "Friend", "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "/ip4/127.0.0.1/tcp/5001/p2p/QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H", "friend-user-id", "some-enc-pub-key", "friend-master-pub-key")

	req := httptest.NewRequest("POST", "/api/v1/admin/keys/reset-identity", bytes.NewReader([]byte("{}")))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	expectedMsg := "P2P identity reset successfully. Revocation and reconnect requests sent to all contacts. Restart required."
	if resp["message"] != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, resp["message"])
	}
	if resp["mnemonic"] == "" {
		t.Errorf("expected generated mnemonic in response, got empty string")
	}

	// Verify contacts are NOT wiped
	contacts, err := database.GetContacts("u1")
	if err != nil {
		t.Fatalf("failed to get contacts: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != "c1" {
		t.Errorf("expected contact c1 to be preserved, got %d contacts", len(contacts))
	}

	// Verify indices are reset to 1
	if cfg.Keys.IdentityKeyIndex != 1 || cfg.Keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected indices to be reset to 1 in config, got identity=%d, encryption=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	idIdx, _ := os.ReadFile(paths.IdentityIndexPath)
	encIdx, _ := os.ReadFile(paths.EncryptionIndexPath)
	if string(idIdx) != "1" || string(encIdx) != "1" {
		t.Errorf("expected indices to be reset to '1' on disk, got identity=%s, encryption=%s", string(idIdx), string(encIdx))
	}
}

func TestAdminRotateKeys_IdentityOnly(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Pre-populate settings and contact
	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.Keys.IdentityKeyIndex = 1
	cfg.Keys.Mnemonic = mnemonic

	// We must write dummy keys first so keymanager can do things
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	cfg.MnemonicStore.Set(mnemonic)

	body, _ := json.Marshal(map[string]bool{
		"rotate_encryption": false,
		"rotate_identity":   true,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	val, _ := os.ReadFile(paths.IdentityIndexPath)
	if string(val) != "2" {
		t.Errorf("expected identity key index on disk to be updated to 2, got %q", string(val))
	}
	if cfg.Keys.IdentityKeyIndex != 2 {
		t.Errorf("expected identity key index in config to be updated to 2, got %d", cfg.Keys.IdentityKeyIndex)
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

func TestAdminUnlockAndStatus(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create admin user and get session cookie.
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Generate a fresh mnemonic to test unlocking
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}
	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		t.Fatalf("failed to marshal master public key: %v", err)
	}

	// Update cfg with the master public key matching our test mnemonic
	cfg.Keys.MasterPublicKeyPEM = masterPubPEM

	// 1. Initial Status check - should be locked
	{
		req := httptest.NewRequest("GET", "/api/v1/admin/status", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status code: %d", rr.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["unlocked"] != false {
			t.Errorf("expected unlocked=false, got %v", resp["unlocked"])
		}
		if int(resp["identity_index"].(float64)) != 1 {
			t.Errorf("expected identity_index=1, got %v", resp["identity_index"])
		}
	}

	// 2. Unlock with Mismatch Mnemonic
	{
		mismatchMnemonic, _ := crypto.GenerateMnemonic()
		body, _ := json.Marshal(map[string]string{"mnemonic": mismatchMnemonic})
		req := httptest.NewRequest("POST", "/api/v1/admin/unlock", bytes.NewReader(body))
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rr.Code)
		}
	}

	// 3. Unlock with Invalid Mnemonic Phrase
	{
		body, _ := json.Marshal(map[string]string{"mnemonic": "invalid mnemonic"})
		req := httptest.NewRequest("POST", "/api/v1/admin/unlock", bytes.NewReader(body))
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rr.Code)
		}
	}

	// 4. Success Unlock
	{
		body, _ := json.Marshal(map[string]string{"mnemonic": mnemonic})
		req := httptest.NewRequest("POST", "/api/v1/admin/unlock", bytes.NewReader(body))
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["message"] != "system unlocked successfully" {
			t.Errorf("expected unlock message, got %q", resp["message"])
		}

		if cfg.MnemonicStore.Get() != mnemonic {
			t.Errorf("expected MnemonicStore to be set to %q, got %q", mnemonic, cfg.MnemonicStore.Get())
		}
	}

	// 5. Status check - should be unlocked
	{
		req := httptest.NewRequest("GET", "/api/v1/admin/status", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status code: %d", rr.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["unlocked"] != true {
			t.Errorf("expected unlocked=true, got %v", resp["unlocked"])
		}
	}
}

func TestAdminRotateKeys_Both(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	_ = os.MkdirAll(filepath.Join(cfg.Root, "core"), 0750)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	// Pre-populate settings and contact
	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.Keys.IdentityKeyIndex = 1
	cfg.Keys.EncryptionKeyIndex = 1
	cfg.Keys.Mnemonic = mnemonic

	// We must write dummy keys first so keymanager can do things
	err := keymanager.RecoverAll(keymanager.NewKeyPaths(cfg.Root), mnemonic, 1, 1)
	if err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	cfg.MnemonicStore.Set(mnemonic)

	body, _ := json.Marshal(map[string]bool{
		"rotate_encryption": true,
		"rotate_identity":   true,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	paths := keymanager.NewKeyPaths(cfg.Root)
	idVal, _ := os.ReadFile(paths.IdentityIndexPath)
	encVal, _ := os.ReadFile(paths.EncryptionIndexPath)
	if string(idVal) != "2" || string(encVal) != "2" {
		t.Errorf("expected identity and encryption indices on disk to be 2, got identity=%s, encryption=%s", string(idVal), string(encVal))
	}
	if cfg.Keys.IdentityKeyIndex != 2 || cfg.Keys.EncryptionKeyIndex != 2 {
		t.Errorf("expected indices in config to be 2, got identity=%d, encryption=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}
}

func TestAdminRotateKeys_NoneSelected(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	mnemonic, _ := crypto.GenerateMnemonic()
	cfg.MnemonicStore.Set(mnemonic)

	body, _ := json.Marshal(map[string]bool{
		"rotate_encryption": false,
		"rotate_identity":   false,
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/keys/rotate", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}





