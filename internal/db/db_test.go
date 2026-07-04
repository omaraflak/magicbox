package db

import (
	"path/filepath"
	"testing"
	"time"
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

func TestEnqueueAndGetPendingMessages(t *testing.T) {
	db := setupTestDB(t)

	// Create a user and contact first (foreign key for join).
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")

	// Enqueue a message.
	err := db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("new-key-hex"), 5)
	if err != nil {
		t.Fatalf("EnqueueMessage failed: %v", err)
	}

	// Get pending messages.
	msgs, err := db.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 pending message, got %d", len(msgs))
	}

	m := msgs[0]
	if m.ID != "msg-1" {
		t.Errorf("expected id msg-1, got %s", m.ID)
	}
	if m.AppID != "system:key-update" {
		t.Errorf("expected app_id system:key-update, got %s", m.AppID)
	}
	if string(m.Payload) != "new-key-hex" {
		t.Errorf("expected payload new-key-hex, got %s", string(m.Payload))
	}
	if m.Multiaddr != "/ip4/127.0.0.1/tcp/4001/p2p/peer-123" {
		t.Errorf("expected joined multiaddr, got %s", m.Multiaddr)
	}
	if m.EncPubKey != "enc-pub-hex" {
		t.Errorf("expected joined enc_pub_key, got %s", m.EncPubKey)
	}
	if m.TargetUserID != "bob-uid" {
		t.Errorf("expected joined target_user_id, got %s", m.TargetUserID)
	}
	if m.Attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", m.Attempts)
	}
	if m.MaxAttempts != 5 {
		t.Errorf("expected max_attempts 5, got %d", m.MaxAttempts)
	}
}

func TestDeleteMessage(t *testing.T) {
	db := setupTestDB(t)
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	if err := db.DeleteMessage("msg-1"); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}

	msgs, _ := db.GetPendingMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestIncrementMessageAttempts(t *testing.T) {
	db := setupTestDB(t)
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	// Set next retry to far future.
	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	if err := db.IncrementMessageAttempts("msg-1", futureTime); err != nil {
		t.Fatalf("IncrementMessageAttempts failed: %v", err)
	}

	// Should not appear in pending (next_retry_at is in the future).
	msgs, _ := db.GetPendingMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 pending messages (future retry), got %d", len(msgs))
	}
}

func TestCleanExpiredMessages(t *testing.T) {
	db := setupTestDB(t)
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 1)

	// Increment past max_attempts.
	db.IncrementMessageAttempts("msg-1", time.Now().UTC().Format(time.RFC3339))

	deleted, err := db.CleanExpiredMessages()
	if err != nil {
		t.Fatalf("CleanExpiredMessages failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestGetPendingMessages_DeletedContactExcluded(t *testing.T) {
	db := setupTestDB(t)
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	// Delete the contact.
	db.DeleteContact("contact-1", "user-1")

	// Message should not appear (JOIN excludes deleted contacts).
	msgs, _ := db.GetPendingMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after contact deleted, got %d", len(msgs))
	}
}
