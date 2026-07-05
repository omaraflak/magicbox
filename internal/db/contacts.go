package db

import (
	"database/sql"
	"time"
)

// Contact represents a federated P2P contact.
type Contact struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	PeerID       string `json:"peer_id"`
	Multiaddr    string `json:"multiaddr"`
	TargetUserID string `json:"target_user_id"`
	EncPubKey    string `json:"enc_pub_key"`
	MasterPubKey string `json:"master_pub_key"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// GetContacts returns all contacts for the given user.
func (d *DB) GetContacts(userID string) ([]Contact, error) {
	rows, err := d.conn.Query(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, status, created_at FROM contacts WHERE user_id = ? ORDER BY display_name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContacts(rows)
}

// GetContactByID fetches a contact by ID and owner userID.
func (d *DB) GetContactByID(id string, userID string) (*Contact, error) {
	row := d.conn.QueryRow(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, status, created_at FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
	return scanContact(row)
}

// GetContactByPeerID fetches a contact by the remote peer's libp2p peer ID and the owning user ID.
func (d *DB) GetContactByPeerID(userID, peerID string) (*Contact, error) {
	row := d.conn.QueryRow(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, status, created_at FROM contacts WHERE user_id = ? AND peer_id = ?`, userID, peerID)
	return scanContact(row)
}

// AddContact inserts a new contact into the database.
func (d *DB) AddContact(id, userID, displayName, peerID, multiaddr, targetUserID, encPubKey, masterPubKey string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO contacts (id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, displayName, peerID, multiaddr, targetUserID, encPubKey, masterPubKey, now,
	)
	return err
}

// DeleteContact deletes a contact by ID and owner userID.
func (d *DB) DeleteContact(id string, userID string) error {
	_, err := d.conn.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	_, _ = d.conn.Exec(`DELETE FROM message_queue WHERE contact_id = ?`, id)
	return nil
}

// UpdateContactEncPubKey updates the encryption public key for a contact.
func (d *DB) UpdateContactEncPubKey(id, encPubKey string) error {
	_, err := d.conn.Exec(`UPDATE contacts SET enc_pub_key = ? WHERE id = ?`, encPubKey, id)
	return err
}

// UpdateContactIdentity updates a contact's peer ID, multiaddr, and encryption public key.
func (d *DB) UpdateContactIdentity(id, peerID, multiaddr, encPubKey string) error {
	_, err := d.conn.Exec(
		`UPDATE contacts SET peer_id = ?, multiaddr = ?, enc_pub_key = ? WHERE id = ?`,
		peerID, multiaddr, encPubKey, id,
	)
	return err
}

// WipeAllContactsAndRequests deletes all contacts, contact requests, and queued messages.
func (d *DB) WipeAllContactsAndRequests() error {
	if _, err := d.conn.Exec(`DELETE FROM contacts`); err != nil {
		return err
	}
	if _, err := d.conn.Exec(`DELETE FROM contact_requests`); err != nil {
		return err
	}
	if _, err := d.conn.Exec(`DELETE FROM message_queue`); err != nil {
		return err
	}
	return nil
}

// GetAllContacts returns all contacts in the system.
func (d *DB) GetAllContacts() ([]Contact, error) {
	rows, err := d.conn.Query(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, status, created_at FROM contacts ORDER BY display_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContacts(rows)
}

// GetContactByTargetUserID fetches a contact by its owner's userID and the target remote userID.
func (d *DB) GetContactByTargetUserID(userID, targetUserID string) (*Contact, error) {
	row := d.conn.QueryRow(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, master_pub_key, status, created_at FROM contacts WHERE user_id = ? AND target_user_id = ?`, userID, targetUserID)
	return scanContact(row)
}

// UpdateContactFromRequest updates the contact's peer ID, multiaddress, encryption public key, and master public key, resetting status to 'active'.
func (d *DB) UpdateContactFromRequest(contactID, peerID, multiaddr, encPubKey, masterPubKey string) error {
	_, err := d.conn.Exec(
		`UPDATE contacts SET peer_id = ?, multiaddr = ?, enc_pub_key = ?, master_pub_key = ?, status = 'active' WHERE id = ?`,
		peerID, multiaddr, encPubKey, masterPubKey, contactID,
	)
	return err
}

// UpdateContactStatus updates a contact's status.
func (d *DB) UpdateContactStatus(contactID string, status string) error {
	_, err := d.conn.Exec(`UPDATE contacts SET status = ? WHERE id = ?`, status, contactID)
	return err
}

func scanContact(row RowScanner) (*Contact, error) {
	var c Contact
	err := row.Scan(&c.ID, &c.UserID, &c.DisplayName, &c.PeerID, &c.Multiaddr, &c.TargetUserID, &c.EncPubKey, &c.MasterPubKey, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanContacts(rows *sql.Rows) ([]Contact, error) {
	var contacts []Contact
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, *c)
	}
	return contacts, rows.Err()
}

