package db

import (
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	tempDB := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(tempDB)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := database.Migrate(); err != nil {
		database.conn.Close()
		t.Fatalf("Migrate failed: %v", err)
	}
	return database
}

func TestCreateAndGetUser_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	username := "omar"
	passwordHash := "$2a$12$somehashedpasswordhere"
	isAdmin := true

	if err := db.CreateUser(userID, username, passwordHash, isAdmin); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected user to be found, got nil")
	}
	if user.ID != userID || user.Username != username || user.IsAdmin != isAdmin {
		t.Errorf("GetUserByUsername returned unexpected user details: %+v", user)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	user, err := db.GetUserByUsername("nonexistent")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %+v", user)
	}
}

func TestInsertAndGetApp_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	// Insert user first to satisfy foreign key constraints
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	app := &App{
		ID:          "app-db-id",
		AppID:       "com.example.drive",
		UserID:      userID,
		Status:      "running",
		RouteSlug:   "drive",
		Image:       "docker.io/library/drive:latest",
		ImageDigest: "sha256:123",
		Version:     "1.0.0",
		ContainerID: "cont-123",
		EntryPort:   9090,
	}

	if err := db.InsertApp(app); err != nil {
		t.Fatalf("InsertApp failed: %v", err)
	}

	fetchedApp, err := db.GetAppByRouteSlugAndUserID(app.RouteSlug, userID)
	if err != nil {
		t.Fatalf("GetAppByRouteSlugAndUserID failed: %v", err)
	}
	if fetchedApp == nil {
		t.Fatal("expected app to be found, got nil")
	}
	if fetchedApp.AppID != app.AppID || fetchedApp.ContainerID != app.ContainerID {
		t.Errorf("fetched app mismatch: expected %+v, got %+v", app, fetchedApp)
	}
}

func TestGetAppByRouteSlugAndUserID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	fetchedApp, err := db.GetAppByRouteSlugAndUserID("missing-slug", "user-123")
	if err != nil {
		t.Fatalf("GetAppByRouteSlugAndUserID failed: %v", err)
	}
	if fetchedApp != nil {
		t.Errorf("expected nil app, got %+v", fetchedApp)
	}
}

func TestAddAndGetContact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	// Insert user first to satisfy foreign key constraints
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contactID := "contact-456"
	contactName := "Alice"
	peerID := "12D3KooWTestPeerID"
	multiaddr := "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeerID"
	targetUserID := "alice-id"

	if err := db.AddContact(contactID, userID, contactName, peerID, multiaddr, targetUserID, "test-enc-pub-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	contacts, err := db.GetContacts(userID)
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].DisplayName != contactName || contacts[0].TargetUserID != targetUserID {
		t.Errorf("unexpected contact details: %+v", contacts[0])
	}
	if contacts[0].PeerID != peerID {
		t.Errorf("expected PeerID %q, got %q", peerID, contacts[0].PeerID)
	}
}

func TestGetContactByPeerID_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	peerID := "12D3KooWTestPeerID"
	if err := db.AddContact("contact-1", userID, "Alice", peerID, "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeerID", "alice-id", "enc-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	contact, err := db.GetContactByPeerID(userID, peerID)
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be found, got nil")
	}
	if contact.PeerID != peerID {
		t.Errorf("expected PeerID %q, got %q", peerID, contact.PeerID)
	}
	if contact.DisplayName != "Alice" {
		t.Errorf("expected DisplayName %q, got %q", "Alice", contact.DisplayName)
	}
}

func TestGetContactByPeerID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contact, err := db.GetContactByPeerID(userID, "nonexistent-peer")
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact != nil {
		t.Errorf("expected nil contact, got %+v", contact)
	}
}

func TestSystemSettings_GetAndSet(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	// Verify default values seeded by migration
	idIdx, err := db.GetSystemSetting(SettingIdentityKeyIndex)
	if err != nil {
		t.Fatalf("failed to get default identity key index: %v", err)
	}
	if idIdx != "0" {
		t.Errorf("expected default identity key index to be '0', got %q", idIdx)
	}

	encIdx, err := db.GetSystemSetting(SettingEncryptionKeyIndex)
	if err != nil {
		t.Fatalf("failed to get default encryption key index: %v", err)
	}
	if encIdx != "0" {
		t.Errorf("expected default encryption key index to be '0', got %q", encIdx)
	}

	// Update setting
	err = db.SetSystemSetting(SettingIdentityKeyIndex, "5")
	if err != nil {
		t.Fatalf("SetSystemSetting failed: %v", err)
	}

	idIdx, err = db.GetSystemSetting(SettingIdentityKeyIndex)
	if err != nil {
		t.Fatalf("GetSystemSetting failed: %v", err)
	}
	if idIdx != "5" {
		t.Errorf("expected identity key index to be updated to '5', got %q", idIdx)
	}

	// Set new setting
	err = db.SetSystemSetting("custom_config", "hello_world")
	if err != nil {
		t.Fatalf("SetSystemSetting failed: %v", err)
	}

	val, err := db.GetSystemSetting("custom_config")
	if err != nil {
		t.Fatalf("GetSystemSetting failed: %v", err)
	}
	if val != "hello_world" {
		t.Errorf("expected 'hello_world', got %q", val)
	}
}
