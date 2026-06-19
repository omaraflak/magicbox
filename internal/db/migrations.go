package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Migrate creates all required tables and seeds initial data.
func (d *DB) Migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			user_id TEXT NOT NULL REFERENCES users(id),
			status TEXT NOT NULL DEFAULT 'stopped',
			route_slug TEXT NOT NULL,
			image TEXT NOT NULL,
			image_digest TEXT,
			version TEXT,
			container_id TEXT,
			host TEXT,
			installed_at TEXT NOT NULL,
			updated_at TEXT,
			UNIQUE(app_id, user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS app_tokens (
			app_id TEXT NOT NULL,
			user_id TEXT NOT NULL REFERENCES users(id),
			token_secret TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (app_id, user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS app_scopes (
			app_id TEXT NOT NULL,
			user_id TEXT NOT NULL REFERENCES users(id),
			scope TEXT NOT NULL,
			granted_at TEXT NOT NULL,
			PRIMARY KEY (app_id, user_id, scope)
		)`,

		`CREATE TABLE IF NOT EXISTS allowed_registries (
			id TEXT PRIMARY KEY,
			prefix TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Incremental schema update: add host column to apps table if it doesn't exist.
	// Ignored if column already exists.
	_, _ = d.conn.Exec(`ALTER TABLE apps ADD COLUMN host TEXT`)

	// Seed default allowed registry.
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO allowed_registries (id, prefix, created_at) VALUES (?, ?, ?)`,
		uuid.NewString(), "docker.io/magicbox/", now,
	)
	if err != nil {
		return fmt.Errorf("failed to seed allowed_registries: %w", err)
	}

	return nil
}
