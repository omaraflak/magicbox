package db

import (
	"testing"
)

func TestInsertRegistry_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.InsertRegistry("reg-1", "docker.io/library/")
	if err != nil {
		t.Fatalf("InsertRegistry failed: %v", err)
	}
}

func TestIsImageAllowed_Allowed(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.InsertRegistry("reg-1", "docker.io/library/")
	if err != nil {
		t.Fatalf("InsertRegistry failed: %v", err)
	}

	allowed, err := db.IsImageAllowed("docker.io/library/alpine:latest")
	if err != nil {
		t.Fatalf("IsImageAllowed failed: %v", err)
	}
	if !allowed {
		t.Error("expected image to be allowed")
	}
}

func TestIsImageAllowed_Disallowed(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.InsertRegistry("reg-1", "docker.io/library/")
	if err != nil {
		t.Fatalf("InsertRegistry failed: %v", err)
	}

	allowed, err := db.IsImageAllowed("gcr.io/my-project/alpine:latest")
	if err != nil {
		t.Fatalf("IsImageAllowed failed: %v", err)
	}
	if allowed {
		t.Error("expected image from unallowed registry to be blocked")
	}
}

func TestDeleteRegistry_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.InsertRegistry("reg-1", "docker.io/library/")
	if err != nil {
		t.Fatalf("InsertRegistry failed: %v", err)
	}

	err = db.DeleteRegistry("reg-1")
	if err != nil {
		t.Fatalf("DeleteRegistry failed: %v", err)
	}
}

func TestIsImageAllowed_BlockedAfterDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.InsertRegistry("reg-1", "docker.io/library/")
	if err != nil {
		t.Fatalf("InsertRegistry failed: %v", err)
	}

	err = db.DeleteRegistry("reg-1")
	if err != nil {
		t.Fatalf("DeleteRegistry failed: %v", err)
	}

	allowed, err := db.IsImageAllowed("docker.io/library/alpine:latest")
	if err != nil {
		t.Fatalf("IsImageAllowed failed: %v", err)
	}
	if allowed {
		t.Error("expected image to be blocked after registry prefix is deleted")
	}
}
