package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Volume mapping: logical name → filesystem path

func initDB() {
	dbPath := "/data/app_state/drive.db"
	os.MkdirAll("/data/app_state", 0755)

	var err error
	dbConn, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open sqlite database: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS sent_history (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			path TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'completed',
			sent_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trash (
			id TEXT PRIMARY KEY,
			original_name TEXT NOT NULL,
			original_path TEXT NOT NULL,
			trash_name TEXT NOT NULL,
			deleted_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_folders (
			id TEXT PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS auto_send_targets (
			folder_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			contact_name TEXT NOT NULL,
			PRIMARY KEY (folder_id, contact_id)
		)`,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			log.Fatalf("Failed to run schema queries: %v", err)
		}
	}

	// Safe DB column migration
	_, _ = dbConn.Exec("ALTER TABLE sent_history ADD COLUMN status TEXT NOT NULL DEFAULT 'completed'")
}

func getTrashRecord(trashName string) (origName string, origPath string, err error) {
	err = dbConn.QueryRow("SELECT original_name, original_path FROM trash WHERE trash_name = ?", trashName).Scan(&origName, &origPath)
	return
}

func deleteTrashRecord(trashName string) error {
	_, err := dbConn.Exec("DELETE FROM trash WHERE trash_name = ?", trashName)
	return err
}

func emptyTrashRecords() error {
	_, err := dbConn.Exec("DELETE FROM trash")
	return err
}

func getExpiredTrashRecords(cutoff string) ([]string, error) {
	rows, err := dbConn.Query("SELECT trash_name FROM trash WHERE deleted_at < ?", cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trashNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			trashNames = append(trashNames, name)
		}
	}
	return trashNames, nil
}

func insertSentHistory(id, filename, path, contactID, contactName, status, sentAt string) error {
	_, err := dbConn.Exec("INSERT INTO sent_history (id, filename, path, contact_id, contact_name, status, sent_at) VALUES (?, ?, ?, ?, ?, ?, ?)", id, filename, path, contactID, contactName, status, sentAt)
	return err
}

func updateSentHistoryStatus(id, status string) error {
	_, err := dbConn.Exec("UPDATE sent_history SET status = ? WHERE id = ?", status, id)
	return err
}

func updateSentHistoryStatusAndDate(id, status, sentAt string) error {
	_, err := dbConn.Exec("UPDATE sent_history SET status = ?, sent_at = ? WHERE id = ?", status, sentAt, id)
	return err
}

func getAllTransfers(limit int) ([]TransferRecord, error) {
	query := "SELECT id, filename, path, contact_id, contact_name, status, sent_at FROM sent_history ORDER BY sent_at DESC"
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}
	rows, err := dbConn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []TransferRecord{}
	for rows.Next() {
		var rec TransferRecord
		if err := rows.Scan(&rec.ID, &rec.Filename, &rec.Path, &rec.ContactID, &rec.ContactName, &rec.Status, &rec.SentAt); err == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

func getSentHistoryByFile(filename, path string) ([]TransferRecord, error) {
	rows, err := dbConn.Query("SELECT id, filename, path, contact_id, contact_name, status, sent_at FROM sent_history WHERE filename = ? AND path = ? ORDER BY sent_at DESC", filename, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []TransferRecord{}
	for rows.Next() {
		var rec TransferRecord
		if err := rows.Scan(&rec.ID, &rec.Filename, &rec.Path, &rec.ContactID, &rec.ContactName, &rec.Status, &rec.SentAt); err == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

func countActiveTransfers() (int, error) {
	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM sent_history WHERE status = 'sending'").Scan(&count)
	return count, err
}

func getActiveTransfers(recentFailedCutoff, recentCompletedCutoff string) ([]TransferRecord, error) {
	rows, err := dbConn.Query(`
		SELECT id, filename, path, contact_id, contact_name, status, sent_at 
		FROM sent_history 
		WHERE status = 'sending' 
		   OR (status = 'failed' AND sent_at > ?) 
		   OR (status = 'completed' AND sent_at > ?)
	`, recentFailedCutoff, recentCompletedCutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []TransferRecord
	for rows.Next() {
		var rec TransferRecord
		if err := rows.Scan(&rec.ID, &rec.Filename, &rec.Path, &rec.ContactID, &rec.ContactName, &rec.Status, &rec.SentAt); err == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

type TrashRecord struct {
	Name         string
	OriginalName string
	OriginalPath string
	DeletedAt    string
}

func getAllTrashRecords() ([]TrashRecord, error) {
	rows, err := dbConn.Query("SELECT original_name, original_path, trash_name, deleted_at FROM trash")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []TrashRecord
	for rows.Next() {
		var rec TrashRecord
		if err := rows.Scan(&rec.OriginalName, &rec.OriginalPath, &rec.Name, &rec.DeletedAt); err == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

func insertTrashRecord(id, originalName, originalPath, trashName, deletedAt string) error {
	_, err := dbConn.Exec("INSERT INTO trash (id, original_name, original_path, trash_name, deleted_at) VALUES (?, ?, ?, ?, ?)", id, originalName, originalPath, trashName, deletedAt)
	return err
}

func countAutoSendFolder(path string) (int, error) {
	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM auto_send_folders WHERE path = ?", path).Scan(&count)
	return count, err
}

func getAllAutoSendFolderPaths() ([]string, error) {
	rows, err := dbConn.Query("SELECT path FROM auto_send_folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

func updateAutoSendFolderPath(oldPath, newPath string) error {
	_, err := dbConn.Exec("UPDATE auto_send_folders SET path = ? WHERE path = ?", newPath, oldPath)
	return err
}

func updateAutoSendFolderPrefix(destRel string, substrStart int, prefix string) error {
	subQuery := `
		UPDATE auto_send_folders 
		SET path = ? || substr(path, ?) 
		WHERE path LIKE ?
	`
	_, err := dbConn.Exec(subQuery, destRel, substrStart, prefix)
	return err
}



func getAllAutoSendFolders() ([]AutoSendFolderInfo, error) {
	rows, err := dbConn.Query("SELECT id, path, created_at FROM auto_send_folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []AutoSendFolderInfo
	for rows.Next() {
		var f AutoSendFolderInfo
		if err := rows.Scan(&f.ID, &f.Path, &f.CreatedAt); err == nil {
			folders = append(folders, f)
		}
	}
	return folders, nil
}

func getAutoSendTargetsByFolder(folderID string) ([]AutoSendTarget, error) {
	rows, err := dbConn.Query("SELECT contact_id, contact_name FROM auto_send_targets WHERE folder_id = ?", folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []AutoSendTarget
	for rows.Next() {
		var t AutoSendTarget
		if err := rows.Scan(&t.ContactID, &t.ContactName); err == nil {
			targets = append(targets, t)
		}
	}
	return targets, nil
}

func getAutoSendFolderByPath(path string) (id string, createdAt string, err error) {
	err = dbConn.QueryRow("SELECT id, created_at FROM auto_send_folders WHERE path = ?", path).Scan(&id, &createdAt)
	return
}

func getAutoSendFolderIDByPath(path string) (string, error) {
	var id string
	err := dbConn.QueryRow("SELECT id FROM auto_send_folders WHERE path = ?", path).Scan(&id)
	return id, err
}

func countAutoSendFoldersByPrefix(prefix string) (int, error) {
	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM auto_send_folders WHERE path LIKE ?", prefix).Scan(&count)
	return count, err
}

func getAutoSendTargetName(contactID string) (string, error) {
	var name string
	err := dbConn.QueryRow("SELECT contact_name FROM auto_send_targets WHERE contact_id = ?", contactID).Scan(&name)
	return name, err
}

func beginTx() (*sql.Tx, error) {
	return dbConn.Begin()
}

func insertAutoSendFolderTx(tx *sql.Tx, id, path, createdAt string) error {
	_, err := tx.Exec("INSERT INTO auto_send_folders (id, path, created_at) VALUES (?, ?, ?)", id, path, createdAt)
	return err
}

func insertAutoSendTargetTx(tx *sql.Tx, folderID, contactID, contactName string) error {
	_, err := tx.Exec("INSERT INTO auto_send_targets (folder_id, contact_id, contact_name) VALUES (?, ?, ?)", folderID, contactID, contactName)
	return err
}

func deleteAutoSendTargetsTx(tx *sql.Tx, folderID string) error {
	_, err := tx.Exec("DELETE FROM auto_send_targets WHERE folder_id = ?", folderID)
	return err
}

func deleteAutoSendFolderTx(tx *sql.Tx, folderID string) error {
	_, err := tx.Exec("DELETE FROM auto_send_folders WHERE id = ?", folderID)
	return err
}
