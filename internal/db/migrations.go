package db

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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
			peer_id TEXT NOT NULL DEFAULT '',
			multiaddr TEXT NOT NULL,
			target_user_id TEXT NOT NULL DEFAULT '',
			enc_pub_key TEXT NOT NULL DEFAULT '',
			master_pub_key TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			UNIQUE(user_id, peer_id)
		)`,

		`CREATE TABLE IF NOT EXISTS message_queue (
			id TEXT PRIMARY KEY,
			contact_id TEXT NOT NULL,
			app_id TEXT NOT NULL,
			payload BLOB NOT NULL,
			next_retry_at TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 10,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS contact_requests (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			direction TEXT NOT NULL,
			display_name TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			multiaddr TEXT NOT NULL,
			target_user_id TEXT NOT NULL,
			enc_pub_key TEXT NOT NULL,
			master_pub_key TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
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
	_, _ = d.conn.Exec(`ALTER TABLE contacts ADD COLUMN enc_pub_key TEXT NOT NULL DEFAULT ''`)
	_, _ = d.conn.Exec(`ALTER TABLE contacts ADD COLUMN peer_id TEXT NOT NULL DEFAULT ''`)
	_, _ = d.conn.Exec(`ALTER TABLE contacts ADD COLUMN master_pub_key TEXT NOT NULL DEFAULT ''`)
	_, _ = d.conn.Exec(`ALTER TABLE contacts ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`)
	_, _ = d.conn.Exec(`ALTER TABLE contact_requests ADD COLUMN master_pub_key TEXT NOT NULL DEFAULT ''`)

	// Migrate existing contacts: extract peer_id and actual multiaddr from stored invite links.
	d.migrateContactInviteLinks()

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

func (d *DB) migrateContactInviteLinks() {
	rows, err := d.conn.Query(`SELECT id, multiaddr FROM contacts WHERE peer_id = '' AND multiaddr LIKE 'magicbox://invite/%'`)
	if err != nil {
		return
	}
	defer rows.Close()

	type migrationRow struct {
		id, multiaddr string
	}
	var toMigrate []migrationRow
	for rows.Next() {
		var r migrationRow
		if err := rows.Scan(&r.id, &r.multiaddr); err != nil {
			continue
		}
		toMigrate = append(toMigrate, r)
	}
	rows.Close()

	for _, r := range toMigrate {
		b64 := strings.TrimPrefix(r.multiaddr, "magicbox://invite/")
		data, err := base64.URLEncoding.DecodeString(b64)
		if err != nil {
			continue
		}
		var payload struct {
			Multiaddr string `json:"multiaddr"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		peerID := ""
		if idx := strings.LastIndex(payload.Multiaddr, "/p2p/"); idx >= 0 {
			peerID = payload.Multiaddr[idx+5:]
		}
		if peerID != "" {
			d.conn.Exec(`UPDATE contacts SET multiaddr = ?, peer_id = ? WHERE id = ?`, payload.Multiaddr, peerID, r.id)
		}
	}
}
