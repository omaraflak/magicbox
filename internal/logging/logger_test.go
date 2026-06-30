package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLogger_WritesJSONLogs(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("hello world", F("key", "value"))
	logger.Close()

	// Verify log file content
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(entries))
	}

	logPath := filepath.Join(tempDir, entries[0].Name())
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to parse JSON log entry: %v", err)
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}
	if entry["msg"] != "hello world" {
		t.Errorf("expected msg 'hello world', got %v", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Errorf("expected key field to be 'value', got %v", entry["key"])
	}
	if entry["ts"] == nil {
		t.Errorf("expected timestamp 'ts' to be present")
	}
}
