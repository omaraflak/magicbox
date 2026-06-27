package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// --- JSON helpers ---

func handleRestoreTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filename := r.URL.Query().Get("file")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	origName, origPath, err := getTrashRecord(filename)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "trash file not found in database")
		} else {
			writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}

	destDir, err := resolvePath("storage", origPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid original path: "+err.Error())
		return
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create destination folders: "+err.Error())
		return
	}

	srcFullPath := filepath.Join(volumes["trash"], filename)
	destFullPath := filepath.Join(destDir, origName)

	if _, err := os.Stat(destFullPath); !os.IsNotExist(err) {
		writeError(w, http.StatusConflict, "file/folder already exists in restored destination")
		return
	}

	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore file: "+err.Error())
		return
	}

	deleteTrashRecord(filename)

	writeJSON(w, http.StatusOK, map[string]string{"restored": origName})
}

func handleEmptyTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	trashDir := volumes["trash"]
	entries, err := os.ReadDir(trashDir)
	if err == nil {
		for _, entry := range entries {
			fullPath := filepath.Join(trashDir, entry.Name())
			if err := os.RemoveAll(fullPath); err != nil {
				log.Printf("handleEmptyTrash: failed to remove %s: %v", fullPath, err)
			}
		}
	}

	dbErr := emptyTrashRecords()
	if dbErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear database records: "+dbErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "trash emptied successfully"})
}
