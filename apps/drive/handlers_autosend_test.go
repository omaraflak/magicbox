package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAutoSendGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)
	setupMockCoreServer(t)

	// insert a folder
	tx, _ := beginTx()
	insertAutoSendFolderTx(tx, "f1", "path/to/sync", time.Now().Format(time.RFC3339))
	insertAutoSendTargetTx(tx, "f1", "c1", "Alice")
	tx.Commit()

	req := httptest.NewRequest(http.MethodGet, "/autosend?path=path/to/sync", nil)
	rr := httptest.NewRecorder()

	handleAutoSend(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	isAutoSend, ok := resp["is_auto_send"].(bool)
	if !ok || !isAutoSend {
		t.Errorf("expected is_auto_send = true")
	}

	targets := resp["targets"].([]interface{})
	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}
}

func TestHandleAutoSendPost(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)
	setupMockCoreServer(t)

	body := map[string]interface{}{
		"path":        "new/sync/folder",
		"contact_ids": []string{"c1"}, // Maps to Alice via mockCoreServer
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/autosend", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handleAutoSend(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify DB was updated
	folders, _ := getAllAutoSendFolders()
	if len(folders) != 1 || folders[0].Path != "new/sync/folder" {
		t.Errorf("expected 1 folder with path new/sync/folder")
	}

	targets, _ := getAutoSendTargetsByFolder(folders[0].ID)
	if len(targets) != 1 || targets[0].ContactName != "Alice" {
		t.Errorf("expected Alice target, got %v", targets)
	}
}
