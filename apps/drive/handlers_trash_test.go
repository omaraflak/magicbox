package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestEnv(t *testing.T) string {
	tempDir := t.TempDir()
	volumes = map[string]string{
		"storage": tempDir + "/storage",
		"trash":   tempDir + "/trash",
	}
	os.MkdirAll(volumes["storage"], 0755)
	os.MkdirAll(volumes["trash"], 0755)
	return tempDir
}

func TestHandleEmptyTrash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	// create fake file in trash
	trashFile := filepath.Join(volumes["trash"], "fake_123")
	os.WriteFile(trashFile, []byte("data"), 0644)

	// insert to db
	insertTrashRecord("1", "fake.txt", "", "fake_123", time.Now().Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodPost, "/trash/empty", nil)
	rr := httptest.NewRecorder()

	handleEmptyTrash(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify file is gone
	if _, err := os.Stat(trashFile); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be deleted", trashFile)
	}

	// Verify DB is empty
	records, err := getAllTrashRecords()
	if err != nil {
		t.Fatalf("getAllTrashRecords failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 db records, got %d", len(records))
	}
}

func TestHandleRestoreTrash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	// Create fake file in trash
	trashFile := filepath.Join(volumes["trash"], "fake_123")
	os.WriteFile(trashFile, []byte("data"), 0644)

	// Insert into DB
	insertTrashRecord("1", "restored.txt", "subfolder", "fake_123", time.Now().Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodPost, "/trash/restore?file=fake_123", nil)
	rr := httptest.NewRecorder()

	handleRestoreTrash(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify JSON response
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["restored"] != "restored.txt" {
		t.Errorf("expected 'restored.txt', got %s", resp["restored"])
	}

	// Verify file is moved
	destFile := filepath.Join(volumes["storage"], "subfolder", "restored.txt")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Errorf("expected file to be restored to %s", destFile)
	}

	// Verify DB record is deleted
	records, _ := getAllTrashRecords()
	if len(records) != 0 {
		t.Errorf("expected DB record to be deleted, found %d", len(records))
	}
}
