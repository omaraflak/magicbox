package cron

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicbox/core/internal/logging"
)

func TestCleanTransit_RemovesOldFiles(t *testing.T) {
	tempDir := t.TempDir()
	transitDir := filepath.Join(tempDir, "transit")
	_ = os.MkdirAll(transitDir, 0750)

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create a new file (not older than 24h)
	newFile := filepath.Join(transitDir, "new_file.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	// Create an old file (older than 24h)
	oldFile := filepath.Join(transitDir, "old_file.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatalf("failed to write old file: %v", err)
	}
	
	// Set modification time to 25 hours ago
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to change times on old file: %v", err)
	}

	cleanTransit(transitDir, logger)

	// Verify old file was deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old file to be deleted, but it exists")
	}

	// Verify new file still exists
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("expected new file to exist, but got error: %v", err)
	}
}
