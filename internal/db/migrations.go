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
			name TEXT NOT NULL DEFAULT '',
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

		`CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			display_name TEXT NOT NULL,
			multiaddr TEXT NOT NULL,
			target_user_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(user_id, multiaddr)
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Try to add 'name' column for existing installations
	_, _ = d.conn.Exec("ALTER TABLE apps ADD COLUMN name TEXT NOT NULL DEFAULT ''")

	// Incremental schema updates: add columns if they don't exist.
	// Ignored if column already exists (SQLite silently errors on duplicate ALTER).
	_, _ = d.conn.Exec(`ALTER TABLE apps ADD COLUMN host TEXT`)
	_, _ = d.conn.Exec(`ALTER TABLE apps ADD COLUMN entry_port INTEGER DEFAULT 9090`)
	_, _ = d.conn.Exec(`ALTER TABLE apps ADD COLUMN webhook_path TEXT DEFAULT '/internal/magicbox-webhook'`)
	_, _ = d.conn.Exec(`ALTER TABLE contacts ADD COLUMN target_user_id TEXT NOT NULL DEFAULT ''`)

	// Seed default allowed registry.
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO allowed_registries (id, prefix, created_at) VALUES (?, ?, ?)`,
		uuid.NewString(), "docker.io/omaraflak/", now,
	)
	if err != nil {
		return fmt.Errorf("failed to seed allowed_registries: %w", err)
	}

	return nil
}
