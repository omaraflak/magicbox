package db

import (
	"database/sql"
	"fmt"
	"time"
)

// InsertP2PPairingToken stores a new authorized client pairing token.
func (d *DB) InsertP2PPairingToken(token string) error {
	query := `INSERT INTO p2p_pairing_tokens (token, created_at) VALUES (?, ?)`
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(query, token, now)
	if err != nil {
		return fmt.Errorf("failed to insert p2p token: %w", err)
	}
	return nil
}

// IsValidP2PPairingToken checks if the pairing token exists and is valid.
func (d *DB) IsValidP2PPairingToken(token string) (bool, error) {
	query := `SELECT count(*) FROM p2p_pairing_tokens WHERE token = ?`
	var count int
	err := d.conn.QueryRow(query, token).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to query p2p token validity: %w", err)
	}
	return count > 0, nil
}
