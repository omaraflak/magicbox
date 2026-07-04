package db

import (
	"strings"
	"time"
)

// Registry represents an allowed container image registry prefix.
type Registry struct {
	ID        string
	Prefix    string
	CreatedAt string
}

// InsertRegistry adds an allowed registry prefix.
func (d *DB) InsertRegistry(id, prefix string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO allowed_registries (id, prefix, created_at) VALUES (?, ?, ?)`,
		id, prefix, now,
	)
	return err
}

// ListRegistries returns all allowed registries.
func (d *DB) ListRegistries() ([]Registry, error) {
	rows, err := d.conn.Query(`SELECT id, prefix, created_at FROM allowed_registries`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registries []Registry
	for rows.Next() {
		var r Registry
		if err := rows.Scan(&r.ID, &r.Prefix, &r.CreatedAt); err != nil {
			return nil, err
		}
		registries = append(registries, r)
	}
	return registries, rows.Err()
}

// DeleteRegistry removes an allowed registry by ID.
func (d *DB) DeleteRegistry(id string) error {
	_, err := d.conn.Exec(`DELETE FROM allowed_registries WHERE id = ?`, id)
	return err
}

// IsImageAllowed checks if an image is from an allowed registry.
func (d *DB) IsImageAllowed(image string) (bool, error) {
	registries, err := d.ListRegistries()
	if err != nil {
		return false, err
	}
	for _, r := range registries {
		if strings.HasPrefix(image, r.Prefix) {
			return true, nil
		}
	}
	return false, nil
}
