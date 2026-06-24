package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB initializes a clean, isolated SQLite test DB and overrides volumes
func setupTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_drive.db")
	
	var err error
	dbConn, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
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
		if _, err := dbConn.Exec(q); err != nil {
			t.Fatalf("Failed schema query: %v", err)
		}
	}

	// Override volumes to use isolated temp subdirectories
	volumes = map[string]string{
		"storage": filepath.Join(tempDir, "storage"),
		"trash":   filepath.Join(tempDir, "trash"),
	}
	os.MkdirAll(volumes["storage"], 0755)
	os.MkdirAll(volumes["trash"], 0755)
}

func TestResolveUniqueFilename(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	// Case 1: No clash
	name1 := resolveUniqueFilename(dir, "photo.jpg")
	if name1 != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %s", name1)
	}

	// Case 2: Direct clash
	err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	name2 := resolveUniqueFilename(dir, "photo.jpg")
	if name2 != "photo (1).jpg" {
		t.Errorf("expected photo (1).jpg, got %s", name2)
	}

	// Case 3: Multiple clashes
	err = os.WriteFile(filepath.Join(dir, "photo (1).jpg"), []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	name3 := resolveUniqueFilename(dir, "photo.jpg")
	if name3 != "photo (2).jpg" {
		t.Errorf("expected photo (2).jpg, got %s", name3)
	}
}

func TestManualSendSingleFile(t *testing.T) {
	setupTestDB(t)
	
	// Create nested directory and target file
	targetDir := filepath.Join(volumes["storage"], "images/spain")
	os.MkdirAll(targetDir, 0755)
	err := os.WriteFile(filepath.Join(targetDir, "image.png"), []byte("dummy-png"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/files/send?volume=storage&path=images/spain&file=image.png&contact_id=c1", nil)
	rr := httptest.NewRecorder()

	handleSendFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected send code 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Query database to inspect tasks mapped
	rows, err := dbConn.Query("SELECT filename, path, contact_id FROM sent_history")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected one record in sent_history, got none")
	}

	var filename, path, contactID string
	if err := rows.Scan(&filename, &path, &contactID); err != nil {
		t.Fatal(err)
	}

	// Single files should go directly to the root of Received (path = "")
	if filename != "image.png" || path != "" || contactID != "c1" {
		t.Errorf("unexpected database entry: filename=%q path=%q contactID=%q", filename, path, contactID)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestManualSendFolder(t *testing.T) {
	setupTestDB(t)

	// Create nested folder structures
	targetFolder := filepath.Join(volumes["storage"], "images/spain")
	os.MkdirAll(filepath.Join(targetFolder, "madrid"), 0755)
	
	err := os.WriteFile(filepath.Join(targetFolder, "barcelona.jpg"), []byte("barcelona"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(targetFolder, "madrid/hotel.png"), []byte("hotel"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/files/send?volume=storage&path=images&file=spain&contact_id=c2", nil)
	rr := httptest.NewRecorder()

	handleSendFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Check mapped path rows in DB
	rows, err := dbConn.Query("SELECT filename, path FROM sent_history ORDER BY filename ASC")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Row 1: barcelona.jpg
	if !rows.Next() {
		t.Fatal("expected first row")
	}
	var fname1, path1 string
	if err := rows.Scan(&fname1, &path1); err != nil {
		t.Fatal(err)
	}
	if fname1 != "barcelona.jpg" || path1 != "spain" {
		t.Errorf("unexpected first entry: filename=%q path=%q", fname1, path1)
	}

	// Row 2: hotel.png
	if !rows.Next() {
		t.Fatal("expected second row")
	}
	var fname2, path2 string
	if err := rows.Scan(&fname2, &path2); err != nil {
		t.Fatal(err)
	}
	if fname2 != "hotel.png" || path2 != "spain/madrid" {
		t.Errorf("unexpected second entry: filename=%q path=%q", fname2, path2)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestSendExistingFilesAutoSend(t *testing.T) {
	setupTestDB(t)

	// Create synced files
	targetDir := filepath.Join(volumes["storage"], "photos/vacation")
	os.MkdirAll(targetDir, 0755)
	err := os.WriteFile(filepath.Join(targetDir, "lake.jpg"), []byte("lake"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	sendExistingFiles("photos/vacation", []string{"contact-xyz"})

	// Inspect database
	var filename, path, contactID string
	err = dbConn.QueryRow("SELECT filename, path, contact_id FROM sent_history").Scan(&filename, &path, &contactID)
	if err != nil {
		t.Fatalf("Failed to query DB: %v", err)
	}

	// Path under Received must be relative to photos (photos/vacation -> vacation)
	if filename != "lake.jpg" || path != "vacation" || contactID != "contact-xyz" {
		t.Errorf("unexpected auto-send sync entry: filename=%q path=%q contactID=%q", filename, path, contactID)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestTriggerAutoSendToContacts(t *testing.T) {
	setupTestDB(t)

	// Create new upload target
	targetDir := filepath.Join(volumes["storage"], "photos/vacation/subpath")
	os.MkdirAll(targetDir, 0755)
	err := os.WriteFile(filepath.Join(targetDir, "sunset.jpg"), []byte("sunset"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	targets := []AutoSendTarget{
		{ContactID: "c-auto", ContactName: "Auto Contact"},
	}

	triggerAutoSendToContacts("photos/vacation", "photos/vacation/subpath", "sunset.jpg", targets)

	// Query database
	var filename, path, contactID string
	err = dbConn.QueryRow("SELECT filename, path, contact_id FROM sent_history").Scan(&filename, &path, &contactID)
	if err != nil {
		t.Fatalf("Failed to query DB: %v", err)
	}

	// Path under Received must be relative to photos (photos/vacation -> vacation/subpath)
	if filename != "sunset.jpg" || path != "vacation/subpath" || contactID != "c-auto" {
		t.Errorf("unexpected auto-send trigger entry: filename=%q path=%q contactID=%q", filename, path, contactID)
	}
}

func TestGenerateMultiItemPlan(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	// Create sample directory tree
	os.MkdirAll(filepath.Join(dir, "folderA"), 0755)
	os.MkdirAll(filepath.Join(dir, "folderB"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "file1.txt"), make([]byte, 100), 0644)
	_ = os.WriteFile(filepath.Join(dir, "folderA/file2.txt"), make([]byte, 100), 0644)
	_ = os.WriteFile(filepath.Join(dir, "folderB/file3.txt"), make([]byte, 100), 0644)

	plan, items, err := generateMultiItemPlan(dir, []string{"file1.txt", "folderA"}, "archive.zip")
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(plan.Volumes))
	}

	if len(items) != 2 {
		t.Errorf("expected 2 zip items (file1.txt and folderA/file2.txt), got %d", len(items))
	}

	// Check deterministic sorting
	if items[0].ZipPath != "file1.txt" || items[1].ZipPath != "folderA/file2.txt" {
		t.Errorf("unexpected zip item ordering: %v", items)
	}
}
