// Package cron provides background maintenance jobs for Magicbox.
package cron

import (
	"os"
	"path/filepath"
	"time"

	"github.com/magicbox/core/internal/logging"
)

// StartTransitCleaner starts a background goroutine that deletes entries
// in the transit directory older than 24 hours. It runs every hour.
// Returns a stop function that cancels the ticker.
func StartTransitCleaner(root string, logger *logging.Logger) func() {
	transitDir := filepath.Join(root, "transit")
	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan struct{})

	go func() {
		// Run immediately on start.
		cleanTransit(transitDir, logger)

		for {
			select {
			case <-ticker.C:
				cleanTransit(transitDir, logger)
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

// cleanTransit walks the transit directory and removes entries older than 24h.
func cleanTransit(transitDir string, logger *logging.Logger) {
	cutoff := time.Now().Add(-24 * time.Hour)

	entries, err := os.ReadDir(transitDir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Error("transit cleaner: failed to read directory", logging.F("error", err.Error()))
		}
		return
	}

	removed := 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(transitDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				logger.Error("transit cleaner: failed to remove entry",
					logging.F("path", path),
					logging.F("error", err.Error()))
			} else {
				removed++
			}
		}
	}

	if removed > 0 {
		logger.Info("transit cleaner: removed old entries", logging.F("count", removed))
	}
}
