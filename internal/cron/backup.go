package cron

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/magicbox/core/internal/logging"
)

const maxBackups = 7

// StartBackupJob starts a background goroutine that backs up the database
// file every 24 hours. It runs an immediate backup on start.
// Returns a stop function that cancels the ticker.
func StartBackupJob(dbPath, backupDir string, logger *logging.Logger) func() {
	ticker := time.NewTicker(24 * time.Hour)
	done := make(chan struct{})

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		logger.Error("backup: failed to create backup directory", logging.F("error", err.Error()))
	}

	go func() {
		// Immediate first run.
		runBackup(dbPath, backupDir, logger)

		for {
			select {
			case <-ticker.C:
				runBackup(dbPath, backupDir, logger)
			case <-done:
				return
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
	}
}

// runBackup copies the database file to the backup directory with a timestamp,
// then prunes old backups to keep only the newest maxBackups.
func runBackup(dbPath, backupDir string, logger *logging.Logger) {
	timestamp := time.Now().UTC().Format("20060102-150405")
	backupName := fmt.Sprintf("magicbox-%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	if err := copyFile(dbPath, backupPath); err != nil {
		logger.Error("backup: failed to copy database",
			logging.F("source", dbPath),
			logging.F("error", err.Error()))
		return
	}

	logger.Info("backup: database backed up", logging.F("file", backupName))

	// Prune old backups.
	pruneBackups(backupDir, logger)
}

// copyFile copies src to dst atomically.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	return out.Sync()
}

// pruneBackups keeps only the newest maxBackups files matching "magicbox-*.db".
func pruneBackups(backupDir string, logger *logging.Logger) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		logger.Error("backup: failed to read backup directory", logging.F("error", err.Error()))
		return
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "magicbox-") && strings.HasSuffix(entry.Name(), ".db") {
			backups = append(backups, entry.Name())
		}
	}

	if len(backups) <= maxBackups {
		return
	}

	// Sort ascending by name (timestamps ensure chronological order).
	sort.Strings(backups)

	// Remove oldest backups.
	toDelete := backups[:len(backups)-maxBackups]
	for _, name := range toDelete {
		path := filepath.Join(backupDir, name)
		if err := os.Remove(path); err != nil {
			logger.Error("backup: failed to remove old backup",
				logging.F("file", name),
				logging.F("error", err.Error()))
		} else {
			logger.Info("backup: pruned old backup", logging.F("file", name))
		}
	}
}
