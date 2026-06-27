package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleDeleteFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	// Create fake file in storage
	fileName := "test_delete.txt"
	storageFile := filepath.Join(volumes["storage"], fileName)
	os.WriteFile(storageFile, []byte("data"), 0644)

	req := httptest.NewRequest(http.MethodDelete, "/files?volume=storage&path=&file="+fileName, nil)
	rr := httptest.NewRecorder()

	handleDeleteFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify file is soft deleted (moved to trash)
	if _, err := os.Stat(storageFile); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be removed from storage", storageFile)
	}

	// Verify DB record is created
	records, err := getAllTrashRecords()
	if err != nil {
		t.Fatalf("getAllTrashRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 DB record in trash, found %d", len(records))
	}
	if records[0].OriginalName != fileName {
		t.Errorf("expected trash record to have original name %s, got %s", fileName, records[0].OriginalName)
	}

	// Verify file is physically in trash
	trashFile := filepath.Join(volumes["trash"], records[0].Name)
	if _, err := os.Stat(trashFile); os.IsNotExist(err) {
		t.Errorf("expected file to be moved to %s", trashFile)
	}
}

func TestHandleListFiles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	// Create a test file in storage
	os.WriteFile(filepath.Join(volumes["storage"], "file1.txt"), []byte("hello"), 0644)
	
	// Create a test dir in storage
	os.Mkdir(filepath.Join(volumes["storage"], "dir1"), 0755)

	req := httptest.NewRequest(http.MethodGet, "/files?volume=storage&path=", nil)
	rr := httptest.NewRecorder()

	handleListFiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	var files []FileInfo
	err := json.NewDecoder(rr.Body).Decode(&files)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files/folders, got %d", len(files))
	}
}
