// Package db provides database access for Magicbox using SQLite.
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a *sql.DB connection to the SQLite database.
type DB struct {
	conn *sql.DB
}

// Open opens a SQLite database at the given path with WAL mode enabled.
func Open(path string) (*DB, error) {
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}
