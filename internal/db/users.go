package db

import (
	"database/sql"
	"time"
)

// User represents a Magicbox user account.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    string
}

// CreateUser inserts a new user record.
func (d *DB) CreateUser(id, username, passwordHash string, isAdmin bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	adminVal := 0
	if isAdmin {
		adminVal = 1
	}
	_, err := d.conn.Exec(
		`INSERT INTO users (id, username, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, username, passwordHash, adminVal, now,
	)
	return err
}

// UpdateUserPassword updates the password hash for a specific user ID.
func (d *DB) UpdateUserPassword(id, passwordHash string) error {
	_, err := d.conn.Exec(
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		passwordHash, id,
	)
	return err
}

// GetUserByID returns a user by their ID, or (nil, nil) if not found.
func (d *DB) GetUserByID(id string) (*User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

// GetUserByUsername returns a user by username, or (nil, nil) if not found.
func (d *DB) GetUserByUsername(username string) (*User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = ?`, username,
	)
	return scanUser(row)
}

// ListUsers returns all users.
func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.conn.Query(`SELECT id, username, password_hash, is_admin, created_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// DeleteUser removes a user by ID.
func (d *DB) DeleteUser(id string) error {
	_, err := d.conn.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// UserCount returns the total number of users.
func (d *DB) UserCount() (int, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func scanUser(row RowScanner) (*User, error) {
	var u User
	var isAdmin int
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin != 0
	return &u, nil
}
