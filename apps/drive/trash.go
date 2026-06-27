package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Volume mapping: logical name → filesystem path

func startTrashCleaner() {
	// Run once immediately on startup
	go cleanExpiredTrash()

	ticker := time.NewTicker(12 * time.Hour)
	go func() {
		for range ticker.C {
			cleanExpiredTrash()
		}
	}()
}

func cleanExpiredTrash() {
	if dbConn == nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	trashNames, err := getExpiredTrashRecords(cutoff)
	if err != nil {
		log.Printf("cleanExpiredTrash: failed to query: %v", err)
		return
	}

	for _, name := range trashNames {
		path := filepath.Join(volumes["trash"], name)
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			log.Printf("cleanExpiredTrash: failed to remove %s: %v", path, err)
		}
		deleteTrashRecord(name)
	}
}
