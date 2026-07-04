package db

import (
	"testing"
)

func TestInsertApp_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
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
}

func TestGetAppByRouteSlugAndUserID_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
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
