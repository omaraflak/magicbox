package db

import (
	"database/sql"
	"time"
)

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

func scanApp(row RowScanner) (*App, error) {
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
