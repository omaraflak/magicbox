package db

import (
	"testing"
)

func TestCreateUser_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	username := "omar"
	passwordHash := "$2a$12$somehashedpasswordhere"
	isAdmin := true

	if err := db.CreateUser(userID, username, passwordHash, isAdmin); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
}

func TestGetUserByUsername_Success(t *testing.T) {
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
