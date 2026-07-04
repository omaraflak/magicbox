package db

import (
	"database/sql"
	"time"
)

// ContactRequest represents a pending contact request.
type ContactRequest struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	Direction    string `json:"direction"` // "incoming" or "outgoing"
	DisplayName  string `json:"display_name"`
	PeerID       string `json:"peer_id"`
	Multiaddr    string `json:"multiaddr"`
	TargetUserID string `json:"target_user_id"`
	EncPubKey    string `json:"enc_pub_key"`
	CreatedAt    string `json:"created_at"`
}

// InsertContactRequest stores a new contact request.
func (d *DB) InsertContactRequest(id, userID, direction, displayName, peerID, multiaddr, targetUserID, encPubKey string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO contact_requests (id, user_id, direction, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, direction, displayName, peerID, multiaddr, targetUserID, encPubKey, now,
	)
	return err
}

// GetContactRequests returns all contact requests for a user, optionally filtered by direction.
func (d *DB) GetContactRequests(userID, direction string) ([]ContactRequest, error) {
	var rows *sql.Rows
	var err error
	if direction != "" {
		rows, err = d.conn.Query(
			`SELECT id, user_id, direction, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at
			 FROM contact_requests WHERE user_id = ? AND direction = ? ORDER BY created_at DESC`,
			userID, direction,
		)
	} else {
		rows, err = d.conn.Query(
			`SELECT id, user_id, direction, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at
			 FROM contact_requests WHERE user_id = ? ORDER BY created_at DESC`,
			userID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ContactRequest
	for rows.Next() {
		var r ContactRequest
		if err := rows.Scan(&r.ID, &r.UserID, &r.Direction, &r.DisplayName, &r.PeerID, &r.Multiaddr, &r.TargetUserID, &r.EncPubKey, &r.CreatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}
	return requests, rows.Err()
}

// GetContactRequest returns a single contact request by ID for a user.
func (d *DB) GetContactRequest(userID, id string) (*ContactRequest, error) {
	row := d.conn.QueryRow(
		`SELECT id, user_id, direction, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at
		 FROM contact_requests WHERE user_id = ? AND id = ?`,
		userID, id,
	)
	var r ContactRequest
	if err := row.Scan(&r.ID, &r.UserID, &r.Direction, &r.DisplayName, &r.PeerID, &r.Multiaddr, &r.TargetUserID, &r.EncPubKey, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// GetContactRequestByPeerID looks up an outgoing request by peer ID for the accept handler.
func (d *DB) GetContactRequestByPeerID(userID, peerID string) (*ContactRequest, error) {
	row := d.conn.QueryRow(
		`SELECT id, user_id, direction, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at
		 FROM contact_requests WHERE user_id = ? AND peer_id = ? AND direction = 'outgoing'`,
		userID, peerID,
	)
	var r ContactRequest
	if err := row.Scan(&r.ID, &r.UserID, &r.Direction, &r.DisplayName, &r.PeerID, &r.Multiaddr, &r.TargetUserID, &r.EncPubKey, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// DeleteContactRequest removes a contact request and its queue items.
func (d *DB) DeleteContactRequest(userID, id string) error {
	_, err := d.conn.Exec(`DELETE FROM contact_requests WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return err
	}
	_, _ = d.conn.Exec(`DELETE FROM message_queue WHERE contact_id = ?`, id)
	return nil
}
