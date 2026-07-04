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
