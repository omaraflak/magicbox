package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	var err error
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite memory database: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS sent_history (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			path TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'completed',
			sent_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trash (
			id TEXT PRIMARY KEY,
			original_name TEXT NOT NULL,
			original_path TEXT NOT NULL,
			trash_name TEXT NOT NULL,
			deleted_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_folders (
			id TEXT PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_targets (
			folder_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			PRIMARY KEY (folder_id, contact_id)
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("Failed to run schema queries: %v", err)
		}
	}

	// Set global dbConn for testing
	dbConn = db
	return db
}

func TestInsertAndGetTrashRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := insertTrashRecord("1", "test.txt", "some/path", "trash_123", time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insertTrashRecord failed: %v", err)
	}

	origName, origPath, err := getTrashRecord("trash_123")
	if err != nil {
		t.Fatalf("getTrashRecord failed: %v", err)
	}
	if origName != "test.txt" || origPath != "some/path" {
		t.Errorf("expected test.txt and some/path, got %s and %s", origName, origPath)
	}
}

func TestAutoSendFoldersDAO(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tx, err := beginTx()
	if err != nil {
		t.Fatalf("beginTx failed: %v", err)
	}

	err = insertAutoSendFolderTx(tx, "folder1", "path/to/folder", time.Now().Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		t.Fatalf("insertAutoSendFolderTx failed: %v", err)
	}
	
	err = insertAutoSendTargetTx(tx, "folder1", "contact1", "Alice")
	if err != nil {
		tx.Rollback()
		t.Fatalf("insertAutoSendTargetTx failed: %v", err)
	}
	tx.Commit()

	folders, err := getAllAutoSendFolders()
	if err != nil {
		t.Fatalf("getAllAutoSendFolders failed: %v", err)
	}
	if len(folders) != 1 || folders[0].Path != "path/to/folder" {
		t.Errorf("getAllAutoSendFolders expected 1 folder with path/to/folder, got %v", folders)
	}

	targets, err := getAutoSendTargetsByFolder("folder1")
	if err != nil {
		t.Fatalf("getAutoSendTargetsByFolder failed: %v", err)
	}
	if len(targets) != 1 || targets[0].ContactName != "Alice" {
		t.Errorf("expected Alice target, got %v", targets)
	}
}
