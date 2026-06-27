package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleListTransfers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	// Insert test data
	insertSentHistory("trans1", "doc.pdf", "", "contact1", "Alice", "completed", time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	insertSentHistory("trans2", "image.png", "photos", "contact2", "Bob", "failed", time.Now().Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodGet, "/transfers?limit=10", nil)
	rr := httptest.NewRecorder()

	handleListTransfers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var records []TransferRecord
	if err := json.NewDecoder(rr.Body).Decode(&records); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(records))
	}
	
	// They should be sorted by date desc
	if records[0].Filename != "image.png" {
		t.Errorf("expected newest transfer first (image.png), got %s", records[0].Filename)
	}
}

func TestHandleListFileSentHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupTestEnv(t)

	insertSentHistory("t1", "report.txt", "docs", "c1", "Alice", "completed", time.Now().Format(time.RFC3339))
	insertSentHistory("t2", "other.txt", "docs", "c1", "Alice", "completed", time.Now().Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodGet, "/transfers/file?file=report.txt&path=docs", nil)
	rr := httptest.NewRecorder()

	handleListFileTransfers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	var records []TransferRecord
	if err := json.NewDecoder(rr.Body).Decode(&records); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 history record, got %d", len(records))
	}
	if records[0].Filename != "report.txt" {
		t.Errorf("expected report.txt, got %s", records[0].Filename)
	}
}
