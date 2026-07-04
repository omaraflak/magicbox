package db

import (
	"time"
)

// QueuedMessage represents a pending P2P message in the delivery queue.
type QueuedMessage struct {
	ID           string
	ContactID    string
	AppID        string
	Payload      []byte
	NextRetryAt  string
	Attempts     int
	MaxAttempts  int
	CreatedAt    string
	Multiaddr    string
	EncPubKey    string
	TargetUserID string
}

// EnqueueMessage adds a message to the delivery queue.
func (d *DB) EnqueueMessage(id, contactID, appID string, payload []byte, maxAttempts int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO message_queue (id, contact_id, app_id, payload, next_retry_at, attempts, max_attempts, created_at)
		 VALUES (?, ?, ?, ?, ?, 0, ?, ?)`,
		id, contactID, appID, payload, now, maxAttempts, now,
	)
	return err
}

// GetPendingMessages returns all queued messages ready for delivery attempt.
func (d *DB) GetPendingMessages() ([]QueuedMessage, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := d.conn.Query(
		`SELECT mq.id, mq.contact_id, mq.app_id, mq.payload, mq.next_retry_at,
		        mq.attempts, mq.max_attempts, mq.created_at,
		        COALESCE(c.multiaddr, cr.multiaddr) as multiaddr,
		        COALESCE(c.enc_pub_key, cr.enc_pub_key) as enc_pub_key,
		        COALESCE(c.target_user_id, cr.target_user_id) as target_user_id
		 FROM message_queue mq
		 LEFT JOIN contacts c ON mq.contact_id = c.id
		 LEFT JOIN contact_requests cr ON mq.contact_id = cr.id
		 WHERE mq.attempts < mq.max_attempts 
		   AND mq.next_retry_at <= ?
		   AND (c.id IS NOT NULL OR cr.id IS NOT NULL)
		 ORDER BY mq.next_retry_at ASC`,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []QueuedMessage
	for rows.Next() {
		var m QueuedMessage
		if err := rows.Scan(
			&m.ID, &m.ContactID, &m.AppID, &m.Payload, &m.NextRetryAt,
			&m.Attempts, &m.MaxAttempts, &m.CreatedAt,
			&m.Multiaddr, &m.EncPubKey, &m.TargetUserID,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// DeleteMessage removes a message from the queue.
func (d *DB) DeleteMessage(id string) error {
	_, err := d.conn.Exec(`DELETE FROM message_queue WHERE id = ?`, id)
	return err
}

// IncrementMessageAttempts increments the attempt counter and sets the next retry time.
func (d *DB) IncrementMessageAttempts(id string, nextRetryAt string) error {
	_, err := d.conn.Exec(
		`UPDATE message_queue SET attempts = attempts + 1, next_retry_at = ? WHERE id = ?`,
		nextRetryAt, id,
	)
	return err
}

// CleanExpiredMessages removes messages that have exhausted their retry attempts.
func (d *DB) CleanExpiredMessages() (int64, error) {
	result, err := d.conn.Exec(`DELETE FROM message_queue WHERE attempts >= max_attempts`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
