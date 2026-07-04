package db

import (
	"database/sql"
	"strings"
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

// App represents an installed application instance.
type App struct {
	ID          string
	AppID       string
	Name        string
	UserID      string
	Status      string
	RouteSlug   string
	Image       string
	ImageDigest string
	Version     string
	ContainerID string
	Host        string
	EntryPort   int
	WebhookPath string
	InstalledAt string
	UpdatedAt   string
}

// AppToken represents a per-app per-user authentication token.
type AppToken struct {
	AppID       string
	UserID      string
	TokenSecret string
	CreatedAt   string
}

// Contact represents a federated P2P contact.
type Contact struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	PeerID       string `json:"peer_id"`
	Multiaddr    string `json:"multiaddr"`
	TargetUserID string `json:"target_user_id"`
	EncPubKey    string `json:"enc_pub_key"`
	CreatedAt    string `json:"created_at"`
}

// Registry represents an allowed container image registry prefix.
type Registry struct {
	ID        string
	Prefix    string
	CreatedAt string
}

// ---------------------------------------------------------------------------
// User queries
// ---------------------------------------------------------------------------

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
		u, err := scanUserRow(rows)
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

// ---------------------------------------------------------------------------
// App queries
// ---------------------------------------------------------------------------

// InsertApp inserts a new app record.
func (d *DB) InsertApp(app *App) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO apps (id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		app.ID, app.AppID, app.Name, app.UserID, app.Status, app.RouteSlug,
		app.Image, app.ImageDigest, app.Version, app.ContainerID, app.Host,
		app.EntryPort, app.WebhookPath, now, now,
	)
	return err
}

// GetAppByID returns an app by its primary key ID, or (nil, nil) if not found.
func (d *DB) GetAppByID(id string) (*App, error) {
	row := d.conn.QueryRow(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE id = ?`, id,
	)
	return scanApp(row)
}

// GetAppByAppIDAndUserID returns an app by its composite key, or (nil, nil) if not found.
func (d *DB) GetAppByAppIDAndUserID(appID, userID string) (*App, error) {
	row := d.conn.QueryRow(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE app_id = ? AND user_id = ?`, appID, userID,
	)
	return scanApp(row)
}

// GetAppByRouteSlugAndUserID returns an app by its route slug and owner userID, or (nil, nil) if not found.
func (d *DB) GetAppByRouteSlugAndUserID(routeSlug, userID string) (*App, error) {
	row := d.conn.QueryRow(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE route_slug = ? AND user_id = ?`, routeSlug, userID,
	)
	return scanApp(row)
}

// ListAppsByUserID returns all apps owned by a given user.
func (d *DB) ListAppsByUserID(userID string) ([]App, error) {
	rows, err := d.conn.Query(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE user_id = ?`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// ListAppsByAppID returns all app instances with a given app_id (across users).
func (d *DB) ListAppsByAppID(appID string) ([]App, error) {
	rows, err := d.conn.Query(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE app_id = ?`, appID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// UpdateAppStatus updates the status and container ID of an app.
func (d *DB) UpdateAppStatus(id, status, containerID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`UPDATE apps SET status = ?, container_id = ?, updated_at = ? WHERE id = ?`,
		status, containerID, now, id,
	)
	return err
}

// UpdateAppVersion updates the version and image digest of an app.
func (d *DB) UpdateAppVersion(id, version, imageDigest string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`UPDATE apps SET version = ?, image_digest = ?, updated_at = ? WHERE id = ?`,
		version, imageDigest, now, id,
	)
	return err
}

// DeleteApp removes an app by its primary key ID.
func (d *DB) DeleteApp(id string) error {
	_, err := d.conn.Exec(`DELETE FROM apps WHERE id = ?`, id)
	return err
}

// ListRunningApps returns all apps with status 'running'.
func (d *DB) ListRunningApps() ([]App, error) {
	rows, err := d.conn.Query(
		`SELECT id, app_id, name, user_id, status, route_slug, image, image_digest, version, container_id, host, entry_port, webhook_path, installed_at, updated_at
		 FROM apps WHERE status = ?`, "running",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// ---------------------------------------------------------------------------
// Token queries
// ---------------------------------------------------------------------------

// InsertAppToken inserts a new app token.
func (d *DB) InsertAppToken(appID, userID, tokenSecret string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO app_tokens (app_id, user_id, token_secret, created_at) VALUES (?, ?, ?, ?)`,
		appID, userID, tokenSecret, now,
	)
	return err
}

// GetAppToken returns an app token, or (nil, nil) if not found.
func (d *DB) GetAppToken(appID, userID string) (*AppToken, error) {
	row := d.conn.QueryRow(
		`SELECT app_id, user_id, token_secret, created_at FROM app_tokens WHERE app_id = ? AND user_id = ?`,
		appID, userID,
	)
	var t AppToken
	err := row.Scan(&t.AppID, &t.UserID, &t.TokenSecret, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteAppToken removes an app token.
func (d *DB) DeleteAppToken(appID, userID string) error {
	_, err := d.conn.Exec(
		`DELETE FROM app_tokens WHERE app_id = ? AND user_id = ?`,
		appID, userID,
	)
	return err
}

// UpdateAppTokenSecret updates the token secret for an app token.
func (d *DB) UpdateAppTokenSecret(appID, userID, newSecret string) error {
	_, err := d.conn.Exec(
		`UPDATE app_tokens SET token_secret = ? WHERE app_id = ? AND user_id = ?`,
		newSecret, appID, userID,
	)
	return err
}

// ---------------------------------------------------------------------------
// Scope queries
// ---------------------------------------------------------------------------

// InsertAppScope grants a scope to an app for a user.
func (d *DB) InsertAppScope(appID, userID, scope string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO app_scopes (app_id, user_id, scope, granted_at) VALUES (?, ?, ?, ?)`,
		appID, userID, scope, now,
	)
	return err
}

// ListAppScopes returns all scopes granted to an app for a user.
func (d *DB) ListAppScopes(appID, userID string) ([]string, error) {
	rows, err := d.conn.Query(
		`SELECT scope FROM app_scopes WHERE app_id = ? AND user_id = ?`,
		appID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		scopes = append(scopes, s)
	}
	return scopes, rows.Err()
}

// DeleteAppScopes removes all scopes for an app and user.
func (d *DB) DeleteAppScopes(appID, userID string) error {
	_, err := d.conn.Exec(
		`DELETE FROM app_scopes WHERE app_id = ? AND user_id = ?`,
		appID, userID,
	)
	return err
}

// ---------------------------------------------------------------------------
// Registry queries
// ---------------------------------------------------------------------------

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
// It fetches all registry prefixes and checks if the image starts with any of them.
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

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(row rowScanner) (*User, error) {
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

func scanUserRow(row rowScanner) (*User, error) {
	var u User
	var isAdmin int
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin != 0
	return &u, nil
}

func scanApp(row rowScanner) (*App, error) {
	var a App
	var imageDigest, version, containerID, host, webhookPath, updatedAt sql.NullString
	var entryPort sql.NullInt64
	err := row.Scan(
		&a.ID, &a.AppID, &a.Name, &a.UserID, &a.Status, &a.RouteSlug,
		&a.Image, &imageDigest, &version, &containerID, &host,
		&entryPort, &webhookPath, &a.InstalledAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.ImageDigest = imageDigest.String
	a.Version = version.String
	a.ContainerID = containerID.String
	a.Host = host.String
	if entryPort.Valid {
		a.EntryPort = int(entryPort.Int64)
	} else {
		a.EntryPort = 9090
	}
	if webhookPath.Valid && webhookPath.String != "" {
		a.WebhookPath = webhookPath.String
	} else {
		a.WebhookPath = "/internal/magicbox-webhook"
	}
	a.UpdatedAt = updatedAt.String
	return &a, nil
}

func scanApps(rows *sql.Rows) ([]App, error) {
	var apps []App
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *a)
	}
	return apps, rows.Err()
}

// GetContacts returns all contacts for the given user.
func (d *DB) GetContacts(userID string) ([]Contact, error) {
	rows, err := d.conn.Query(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at FROM contacts WHERE user_id = ? ORDER BY display_name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.UserID, &c.DisplayName, &c.PeerID, &c.Multiaddr, &c.TargetUserID, &c.EncPubKey, &c.CreatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// GetContactByID fetches a contact by ID and owner userID.
func (d *DB) GetContactByID(id string, userID string) (*Contact, error) {
	row := d.conn.QueryRow(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
	var c Contact
	err := row.Scan(&c.ID, &c.UserID, &c.DisplayName, &c.PeerID, &c.Multiaddr, &c.TargetUserID, &c.EncPubKey, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetContactByPeerID fetches a contact by the remote peer's libp2p peer ID and the owning user ID.
func (d *DB) GetContactByPeerID(userID, peerID string) (*Contact, error) {
	row := d.conn.QueryRow(`SELECT id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at FROM contacts WHERE user_id = ? AND peer_id = ?`, userID, peerID)
	var c Contact
	err := row.Scan(&c.ID, &c.UserID, &c.DisplayName, &c.PeerID, &c.Multiaddr, &c.TargetUserID, &c.EncPubKey, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// AddContact inserts a new contact into the database.
func (d *DB) AddContact(id, userID, displayName, peerID, multiaddr, targetUserID, encPubKey string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(
		`INSERT INTO contacts (id, user_id, display_name, peer_id, multiaddr, target_user_id, enc_pub_key, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, displayName, peerID, multiaddr, targetUserID, encPubKey, now,
	)
	return err
}

// DeleteContact deletes a contact by ID and owner userID.
func (d *DB) DeleteContact(id string, userID string) error {
	_, err := d.conn.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// UpdateContactEncPubKey updates the encryption public key for a contact.
func (d *DB) UpdateContactEncPubKey(id, encPubKey string) error {
	_, err := d.conn.Exec(`UPDATE contacts SET enc_pub_key = ? WHERE id = ?`, encPubKey, id)
	return err
}

// ---------------------------------------------------------------------------
// System Settings queries
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Message Queue queries
// ---------------------------------------------------------------------------

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
	// Joined from contacts table
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
// Results include joined contact data (multiaddr, enc_pub_key, target_user_id).
// Messages whose contact has been deleted are automatically excluded by the JOIN.
func (d *DB) GetPendingMessages() ([]QueuedMessage, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := d.conn.Query(
		`SELECT mq.id, mq.contact_id, mq.app_id, mq.payload, mq.next_retry_at,
		        mq.attempts, mq.max_attempts, mq.created_at,
		        c.multiaddr, c.enc_pub_key, c.target_user_id
		 FROM message_queue mq
		 JOIN contacts c ON mq.contact_id = c.id
		 WHERE mq.attempts < mq.max_attempts AND mq.next_retry_at <= ?
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

// DeleteMessage removes a message from the queue (typically after successful delivery).
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
// Returns the number of deleted rows.
func (d *DB) CleanExpiredMessages() (int64, error) {
	result, err := d.conn.Exec(`DELETE FROM message_queue WHERE attempts >= max_attempts`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ---------------------------------------------------------------------------
// Contact Request queries
// ---------------------------------------------------------------------------

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
// If direction is empty, returns all requests.
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

// DeleteContactRequest removes a contact request.
func (d *DB) DeleteContactRequest(userID, id string) error {
	_, err := d.conn.Exec(`DELETE FROM contact_requests WHERE user_id = ? AND id = ?`, userID, id)
	return err
}

// UpdateContactIdentity updates a contact's peer ID, multiaddr, and encryption public key.
// Used when they rotate their identity key and broadcast a succession certificate.
func (d *DB) UpdateContactIdentity(id, peerID, multiaddr, encPubKey string) error {
	_, err := d.conn.Exec(
		`UPDATE contacts SET peer_id = ?, multiaddr = ?, enc_pub_key = ? WHERE id = ?`,
		peerID, multiaddr, encPubKey, id,
	)
	return err
}

