package db

import (
	"database/sql"
)

// Constants for system settings keys.
const (
	SettingIdentityKeyIndex   = "identity_key_index"
	SettingEncryptionKeyIndex = "encryption_key_index"
)

// GetSystemSetting retrieves a system-wide setting by key.
func (d *DB) GetSystemSetting(key string) (string, error) {
	var value string
	err := d.conn.QueryRow(`SELECT value FROM system_settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSystemSetting inserts or updates a system-wide setting by key.
func (d *DB) SetSystemSetting(key, value string) error {
	_, err := d.conn.Exec(
		`INSERT INTO system_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
