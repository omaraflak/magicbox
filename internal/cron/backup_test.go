package cron

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicbox/core/internal/logging"
)

func TestRunBackup_CreatesBackup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "magicbox.db")
	backupDir := filepath.Join(tempDir, "backups")

	// Write mock DB content
	expectedContent := []byte("database content")
	if err := os.WriteFile(dbPath, expectedContent, 0644); err != nil {
		t.Fatalf("failed to write mock db: %v", err)
	}

	// Create the backup directory as the StartBackupJob would do
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	runBackup(dbPath, backupDir, logger)

	// Check if a backup file was created
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup entry, got %d", len(entries))
	}

	backupPath := filepath.Join(backupDir, entries[0].Name())
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	if string(data) != string(expectedContent) {
		t.Errorf("backup content mismatch")
	}
}

func TestPruneBackups_KeepsOnlyMaxBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(backupDir, 0750)

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create 10 mock backup files
	for i := 1; i <= 10; i++ {
		filename := fmt.Sprintf("magicbox-20260102-%02d0000.db", i)
		path := filepath.Join(backupDir, filename)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write mock backup file: %v", err)
		}
	}

	pruneBackups(backupDir, logger)

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	// Should only keep maxBackups (7)
	if len(entries) != maxBackups {
		t.Errorf("expected %d backups to remain, got %d", maxBackups, len(entries))
	}

	// The remaining backups should be the newest ones (index 4 to 10)
	// We deleted index 1, 2, 3
	for _, entry := range entries {
		name := entry.Name()
		if name == "magicbox-20260102-010000.db" || name == "magicbox-20260102-020000.db" || name == "magicbox-20260102-030000.db" {
			t.Errorf("expected old backup %s to be pruned", name)
		}
	}
}
