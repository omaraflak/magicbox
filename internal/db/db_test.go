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
	multiaddr := "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H?user_id=alice-id"
	targetUserID := "alice-id"

	if err := db.AddContact(contactID, userID, contactName, multiaddr, targetUserID, "test-enc-pub-key"); err != nil {
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
}
