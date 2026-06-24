package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"io"
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

func TestDeleteFileSoft(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]
	
	err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("important notes"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/files/delete?volume=storage&path=&file=notes.txt", nil)
	rr := httptest.NewRecorder()

	handleDeleteFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify original file no longer exists
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); !os.IsNotExist(err) {
		t.Error("expected original file to be gone, but it still exists")
	}

	// Verify row in database trash table
	var count int
	err = dbConn.QueryRow("SELECT COUNT(*) FROM trash WHERE original_name = 'notes.txt'").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 trash record in db, got %d", count)
	}
}

func TestDeleteFolderSoft(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	targetFolder := filepath.Join(dir, "docs")
	os.MkdirAll(targetFolder, 0755)
	err := os.WriteFile(filepath.Join(targetFolder, "info.txt"), []byte("info"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/files/delete?volume=storage&path=&file=docs", nil)
	rr := httptest.NewRecorder()

	handleDeleteFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify original folder no longer exists
	if _, err := os.Stat(targetFolder); !os.IsNotExist(err) {
		t.Error("expected original folder to be gone, but it still exists")
	}

	// Verify row in database trash table
	var trashName string
	err = dbConn.QueryRow("SELECT trash_name FROM trash WHERE original_name = 'docs'").Scan(&trashName)
	if err != nil {
		t.Fatal(err)
	}

	// Verify trash directory contains renamed folder
	trashPath := filepath.Join(volumes["trash"], trashName)
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		t.Error("expected folder to exist in trash folder, but it was not found")
	}
}

func TestDeleteTrashPermanent(t *testing.T) {
	setupTestDB(t)
	trashDir := volumes["trash"]

	// Create dummy file inside trash folder
	trashName := "dummy_123456"
	err := os.WriteFile(filepath.Join(trashDir, trashName), []byte("to be permanently deleted"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Add trash database record
	_, err = dbConn.Exec("INSERT INTO trash (id, original_name, original_path, trash_name, deleted_at) VALUES (?, ?, ?, ?, ?)",
		"trash-id-xyz", "dummy.txt", "", trashName, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/files/delete?volume=trash&path=&file="+trashName, nil)
	rr := httptest.NewRecorder()

	handleDeleteFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify trash file is permanently gone from filesystem
	if _, err := os.Stat(filepath.Join(trashDir, trashName)); !os.IsNotExist(err) {
		t.Error("expected trash file to be permanently deleted, but it still exists")
	}

	// Verify database record has been pruned
	var count int
	err = dbConn.QueryRow("SELECT COUNT(*) FROM trash WHERE trash_name = ?", trashName).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected trash record to be deleted, but count is %d", count)
	}
}

func TestDownloadSingleFile(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("image-bytes-data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/files/download?volume=storage&path=&file=photo.jpg", nil)
	rr := httptest.NewRecorder()

	handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Check download header
	dispHeader := rr.Header().Get("Content-Disposition")
	expectedHeader := `attachment; filename="photo.jpg"`
	if dispHeader != expectedHeader {
		t.Errorf("expected header %q, got %q", expectedHeader, dispHeader)
	}

	// Check body bytes
	if rr.Body.String() != "image-bytes-data" {
		t.Errorf("expected body 'image-bytes-data', got %q", rr.Body.String())
	}
}

func TestDownloadFolderZipped(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	// Create subfolder and files
	subDir := filepath.Join(dir, "vacation")
	os.MkdirAll(subDir, 0755)
	err := os.WriteFile(filepath.Join(subDir, "lake.jpg"), []byte("lake-data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/files/download?volume=storage&path=&file=vacation", nil)
	rr := httptest.NewRecorder()

	handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Check zip header
	dispHeader := rr.Header().Get("Content-Disposition")
	expectedHeader := `attachment; filename="vacation.zip"`
	if dispHeader != expectedHeader {
		t.Errorf("expected header %q, got %q", expectedHeader, dispHeader)
	}

	// Unpack ZIP archive to verify contents
	zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read response as ZIP: %v", err)
	}

	if len(zipReader.File) != 1 {
		t.Fatalf("expected 1 file in zip, got %d", len(zipReader.File))
	}

	f := zipReader.File[0]
	if f.Name != "vacation/lake.jpg" {
		t.Errorf("expected zip file name 'vacation/lake.jpg', got %q", f.Name)
	}

	rc, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, rc)
	if buf.String() != "lake-data" {
		t.Errorf("expected unzipped content 'lake-data', got %q", buf.String())
	}
}

func TestDownloadMultipleFilesZipped(t *testing.T) {
	setupTestDB(t)
	dir := volumes["storage"]

	err := os.WriteFile(filepath.Join(dir, "f1.txt"), []byte("file1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("file2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/files/download?volume=storage&path=&file=f1.txt&file=f2.txt", nil)
	rr := httptest.NewRecorder()

	handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Unpack ZIP archive to verify contents
	zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read response as ZIP: %v", err)
	}

	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(zipReader.File))
	}

	// Verify f1.txt content
	f1 := zipReader.File[0]
	if f1.Name != "f1.txt" {
		t.Errorf("expected first zip file name 'f1.txt', got %q", f1.Name)
	}

	// Verify f2.txt content
	f2 := zipReader.File[1]
	if f2.Name != "f2.txt" {
		t.Errorf("expected second zip file name 'f2.txt', got %q", f2.Name)
	}
}

