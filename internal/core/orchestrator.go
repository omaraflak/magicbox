package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
)

// TokenGeneratorFunc is a function that generates a signed JWT for an app.
// It accepts (secret, userID, appID, scopes) and returns a signed token string.
type TokenGeneratorFunc func(secret []byte, userID, appID string, scopes []string) (string, error)

// Orchestrator coordinates app lifecycle operations.
type Orchestrator struct {
	DB             *db.DB
	Docker         *docker.Client
	Cfg            *config.Config
	Logger         *logging.Logger
	TokenGenerator TokenGeneratorFunc

	mu                 sync.RWMutex
	pendingPermissions map[string]*PermissionRequest
}

type PermissionRequest struct {
	ID        string   `json:"id"`
	AppID     string   `json:"app_id"`
	AppName   string   `json:"app_name"`
	Scopes    []string `json:"scopes"`
	Reason    string   `json:"reason"`
	Status    string   `json:"status"` // "pending", "approved", "rejected"
	userID    string
	decision  chan bool
}

// NewOrchestrator creates a new Orchestrator with the given dependencies.
func NewOrchestrator(database *db.DB, dockerClient *docker.Client, cfg *config.Config, logger *logging.Logger, tokenGen TokenGeneratorFunc) *Orchestrator {
	return &Orchestrator{
		DB:                 database,
		Docker:             dockerClient,
		Cfg:                cfg,
		Logger:             logger,
		TokenGenerator:     tokenGen,
		pendingPermissions: make(map[string]*PermissionRequest),
	}
}

// Install installs a new app for the given user from a manifest.
func (o *Orchestrator) Install(ctx context.Context, userID string, manifestData []byte) (*db.App, error) {
	// 1. Parse and validate manifest.
	manifest, err := ParseManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	if errs := ValidateManifest(manifest); len(errs) > 0 {
		return nil, fmt.Errorf("manifest validation failed: %s", strings.Join(errs, "; "))
	}

	// 2. Check image allowlist.
	allowed, err := o.DB.IsImageAllowed(manifest.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to check image allowlist: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("image %q is not from an allowed registry", manifest.Image)
	}

	// 3. Check not already installed.
	existing, err := o.DB.GetAppByAppIDAndUserID(manifest.AppID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing app: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("app %q is already installed", manifest.AppID)
	}

	// 4. Get user for username.
	user, err := o.DB.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 5. Generate IDs and token secret.
	appDBID := uuid.NewString()
	tokenSecret, err := generateTokenSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token secret: %w", err)
	}

	// 6. Create host directories.
	appDir := filepath.Join(o.Cfg.Root, "users", user.Username, "apps", manifest.AppID)
	if err := os.MkdirAll(appDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create app directory: %w", err)
	}
	// Ensure shared directories exist.
	for _, scope := range manifest.RequiredScopes {
		volName, _, ok := ScopeToVolumeAccess(scope)
		if ok {
			sharedDir := filepath.Join(o.Cfg.Root, "users", user.Username, "shared", volName)
			if err := os.MkdirAll(sharedDir, 0750); err != nil {
				return nil, fmt.Errorf("failed to create shared directory: %w", err)
			}
		}
	}

	// 7. Insert app record (status=installing), token, and scopes.
	app := &db.App{
		ID:          appDBID,
		AppID:       manifest.AppID,
		Name:        manifest.Name,
		UserID:      userID,
		Status:      "installing",
		RouteSlug:   manifest.RouteSlug,
		Image:       manifest.Image,
		Version:     manifest.Version,
		Host:        manifest.Host,
		EntryPort:   manifest.EntryPort,
		WebhookPath: manifest.WebhookPath,
	}
	if err := o.DB.InsertApp(app); err != nil {
		return nil, fmt.Errorf("failed to insert app: %w", err)
	}
	if err := o.DB.InsertAppToken(manifest.AppID, userID, tokenSecret); err != nil {
		return nil, fmt.Errorf("failed to insert app token: %w", err)
	}
	for _, scope := range manifest.RequiredScopes {
		if err := o.DB.InsertAppScope(manifest.AppID, userID, scope); err != nil {
			return nil, fmt.Errorf("failed to insert app scope: %w", err)
		}
	}

	// Run the remainder of installation (image pull, container creation and start) in the background.
	go func() {
		bgCtx := context.Background()

		if o.Docker == nil {
			o.Logger.Info("test mode: skipping Docker pull and container creation")
			_ = o.DB.UpdateAppStatus(appDBID, "running", "mock-container-id")
			return
		}

		// 8. Pull image.
		digest, err := o.Docker.PullImage(bgCtx, manifest.Image, false)
		if err != nil {
			o.Logger.Error("install: pull image failed", logging.F("app_id", manifest.AppID), logging.F("error", err.Error()))
			o.DB.UpdateAppStatus(appDBID, "error", "failed to pull image: "+err.Error())
			return
		}
		if err := o.DB.UpdateAppVersion(appDBID, manifest.Version, digest); err != nil {
			o.Logger.Error("install: failed to update app version", logging.F("error", err.Error()))
		}

		// 9. Generate app JWT token for the container.
		scopes, err := o.DB.ListAppScopes(manifest.AppID, userID)
		if err != nil {
			o.Logger.Error("install: failed to list scopes", logging.F("error", err.Error()))
			o.DB.UpdateAppStatus(appDBID, "error", "failed to list scopes: "+err.Error())
			return
		}
		appToken, err := o.TokenGenerator([]byte(tokenSecret), userID, manifest.AppID, scopes)
		if err != nil {
			o.Logger.Error("install: failed to generate app token", logging.F("error", err.Error()))
			o.DB.UpdateAppStatus(appDBID, "error", "failed to generate app token: "+err.Error())
			return
		}

		// 10. Build container config and create+start container.
		var volumeMounts []docker.AppVolumeMount
		for _, scope := range manifest.RequiredScopes {
			volName, readOnly, ok := ScopeToVolumeAccess(scope)
			if ok {
				volumeMounts = append(volumeMounts, docker.AppVolumeMount{
					Name:     volName,
					ReadOnly: readOnly,
				})
			}
		}

		containerCfg := &docker.AppContainerConfig{
			AppID:         manifest.AppID,
			AppName:       manifest.Name,
			Image:         manifest.Image,
			EntryPort:     manifest.EntryPort,
			RouteSlug:     manifest.RouteSlug,
			Username:      user.Username,
			UserID:        userID,
			AppToken:      appToken,
			WebhookSecret: tokenSecret,
			CoreURL:       "magicbox_core:50051",
			MagicboxRoot:  o.Cfg.HostRoot,
			VolumeMounts:  volumeMounts,
			MemoryMB:      manifest.ResourceLimits.MemoryMB,
			CPUCores:      manifest.ResourceLimits.CPUCores,
			Host:          manifest.Host,
		}

		containerID, err := o.Docker.CreateAndStartContainer(bgCtx, containerCfg)
		if err != nil {
			o.Logger.Error("install: start container failed", logging.F("app_id", manifest.AppID), logging.F("error", err.Error()))
			o.DB.UpdateAppStatus(appDBID, "error", "failed to start container: "+err.Error())
			return
		}

		// 11. Update status to running.
		if err := o.DB.UpdateAppStatus(appDBID, "running", containerID); err != nil {
			o.Logger.Error("install: failed to update app status", logging.F("error", err.Error()))
		}

		o.Logger.Info("app installed successfully", logging.F("app_id", manifest.AppID), logging.F("user", user.Username))
	}()

	return app, nil
}

// Uninstall removes an installed app and cleans up all resources.
func (o *Orchestrator) Uninstall(ctx context.Context, appDBID string, wipe bool) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}

	user, err := o.DB.GetUserByID(app.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Stop and remove container if present.
	if app.ContainerID != "" && o.Docker != nil {
		_ = o.Docker.StopContainer(ctx, app.ContainerID, 10)
		_ = o.Docker.RemoveContainer(ctx, app.ContainerID)
	}

	// Remove private app directory (NOT shared dirs) if requested.
	if wipe {
		appDir := filepath.Join(o.Cfg.Root, "users", user.Username, "apps", app.AppID)
		os.RemoveAll(appDir)
	}

	// Delete scopes, tokens, and app row.
	o.DB.DeleteAppScopes(app.AppID, app.UserID)
	o.DB.DeleteAppToken(app.AppID, app.UserID)
	o.DB.DeleteApp(appDBID)

	o.Logger.Info("app uninstalled", logging.F("app_id", app.AppID), logging.F("user", user.Username))
	return nil
}

// Stop stops a running app container.
func (o *Orchestrator) Stop(ctx context.Context, appDBID string) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}
	if app.ContainerID == "" {
		return fmt.Errorf("app has no container")
	}

	if o.Docker != nil {
		if err := o.Docker.StopContainer(ctx, app.ContainerID, 10); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	if err := o.DB.UpdateAppStatus(appDBID, "stopped", app.ContainerID); err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	o.Logger.Info("app stopped", logging.F("app_id", app.AppID))
	return nil
}

// Start starts a stopped app container.
func (o *Orchestrator) Start(ctx context.Context, appDBID string) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}

	user, err := o.DB.GetUserByID(app.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// If there's an existing stopped container, remove it and recreate.
	if app.ContainerID != "" && o.Docker != nil {
		_ = o.Docker.RemoveContainer(ctx, app.ContainerID)
	}

	// Regenerate app token.
	token, err := o.DB.GetAppToken(app.AppID, app.UserID)
	if err != nil || token == nil {
		return fmt.Errorf("failed to get app token")
	}
	scopes, err := o.DB.ListAppScopes(app.AppID, app.UserID)
	if err != nil {
		return fmt.Errorf("failed to list scopes: %w", err)
	}
	appJWT, err := o.TokenGenerator([]byte(token.TokenSecret), app.UserID, app.AppID, scopes)
	if err != nil {
		return fmt.Errorf("failed to generate app token: %w", err)
	}

	// Build volume mounts from scopes.
	var volumeMounts []docker.AppVolumeMount
	for _, scope := range scopes {
		volName, readOnly, ok := ScopeToVolumeAccess(scope)
		if ok {
			volumeMounts = append(volumeMounts, docker.AppVolumeMount{
				Name:     volName,
				ReadOnly: readOnly,
			})
		}
	}

	containerCfg := &docker.AppContainerConfig{
		AppID:         app.AppID,
		AppName:       app.AppID, // Use appID as name fallback.
		Image:         app.Image,
		EntryPort:     app.EntryPort,
		RouteSlug:     app.RouteSlug,
		Username:      user.Username,
		UserID:        app.UserID,
		AppToken:      appJWT,
		WebhookSecret: token.TokenSecret,
		CoreURL:       "magicbox_core:50051",
		MagicboxRoot:  o.Cfg.HostRoot,
		VolumeMounts:  volumeMounts,
		Host:          app.Host,
	}

	var containerID string
	if o.Docker != nil {
		var err error
		containerID, err = o.Docker.CreateAndStartContainer(ctx, containerCfg)
		if err != nil {
			o.DB.UpdateAppStatus(appDBID, "error", "")
			return fmt.Errorf("failed to start container: %w", err)
		}
	} else {
		containerID = "mock-container-id"
	}

	if err := o.DB.UpdateAppStatus(appDBID, "running", containerID); err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	o.Logger.Info("app started", logging.F("app_id", app.AppID))
	return nil
}

// Rebuild re-pulls the app's image and recreates its container.
// Used when the image has been updated (e.g. during development) without changing the manifest.
func (o *Orchestrator) Rebuild(ctx context.Context, appDBID string) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}

	var digest string
	if o.Docker != nil {
		var err error
		digest, err = o.Docker.PullImage(ctx, app.Image, true)
		if err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
	} else {
		digest = "mock-digest"
	}

	// Stop and remove the old container if running.
	if app.ContainerID != "" && o.Docker != nil {
		_ = o.Docker.StopContainer(ctx, app.ContainerID, 10)
		_ = o.Docker.RemoveContainer(ctx, app.ContainerID)
	}

	// Update the image digest in the DB.
	if err := o.DB.UpdateAppVersion(appDBID, app.Version, digest); err != nil {
		return fmt.Errorf("failed to update image digest: %w", err)
	}

	// Start a fresh container with the new image.
	if err := o.Start(ctx, appDBID); err != nil {
		return fmt.Errorf("failed to restart after rebuild: %w", err)
	}

	o.Logger.Info("app rebuilt", logging.F("app_id", app.AppID), logging.F("digest", digest))
	return nil
}

// Update updates an app with a new manifest version.
func (o *Orchestrator) Update(ctx context.Context, appDBID string, manifestData []byte) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}

	manifest, err := ParseManifest(manifestData)
	if err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}
	if errs := ValidateManifest(manifest); len(errs) > 0 {
		return fmt.Errorf("manifest validation failed: %s", strings.Join(errs, "; "))
	}

	if manifest.AppID != app.AppID {
		return fmt.Errorf("cannot change app_id during update")
	}

	// Pull new image.
	var digest string
	if o.Docker != nil {
		var err error
		digest, err = o.Docker.PullImage(ctx, manifest.Image, false)
		if err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
	} else {
		digest = "mock-digest"
	}

	// Stop and remove old container.
	if app.ContainerID != "" && o.Docker != nil {
		_ = o.Docker.StopContainer(ctx, app.ContainerID, 10)
		_ = o.Docker.RemoveContainer(ctx, app.ContainerID)
	}

	// Update DB version.
	if err := o.DB.UpdateAppVersion(appDBID, manifest.Version, digest); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	// Restart with new image via Start.
	if err := o.Start(ctx, appDBID); err != nil {
		return fmt.Errorf("failed to restart after update: %w", err)
	}

	o.Logger.Info("app updated", logging.F("app_id", app.AppID), logging.F("version", manifest.Version))
	return nil
}

// RotateToken generates a new token secret for an app and restarts it.
func (o *Orchestrator) RotateToken(ctx context.Context, appDBID string) error {
	app, err := o.DB.GetAppByID(appDBID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("app not found")
	}

	newSecret, err := generateTokenSecret()
	if err != nil {
		return fmt.Errorf("failed to generate new secret: %w", err)
	}

	if err := o.DB.UpdateAppTokenSecret(app.AppID, app.UserID, newSecret); err != nil {
		return fmt.Errorf("failed to update token secret: %w", err)
	}

	// Restart the container with the new token if it was running.
	if app.Status == "running" {
		if app.ContainerID != "" {
			_ = o.Docker.StopContainer(ctx, app.ContainerID, 10)
			_ = o.Docker.RemoveContainer(ctx, app.ContainerID)
		}
		if err := o.Start(ctx, appDBID); err != nil {
			return fmt.Errorf("failed to restart after token rotation: %w", err)
		}
	}

	o.Logger.Info("token rotated", logging.F("app_id", app.AppID))
	return nil
}

// HealthCheck inspects all running containers and marks crashed ones as error.
func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	apps, err := o.DB.ListRunningApps()
	if err != nil {
		return fmt.Errorf("failed to list running apps: %w", err)
	}

	for _, app := range apps {
		if app.ContainerID == "" {
			continue
		}

		status, err := o.Docker.InspectContainer(ctx, app.ContainerID)
		if err != nil {
			o.Logger.Warn("health check: failed to inspect container",
				logging.F("app_id", app.AppID),
				logging.F("error", err.Error()))
			o.DB.UpdateAppStatus(app.ID, "error", app.ContainerID)
			continue
		}

		if !status.Running {
			o.Logger.Warn("health check: container not running",
				logging.F("app_id", app.AppID),
				logging.F("exit_code", status.ExitCode))
			o.DB.UpdateAppStatus(app.ID, "error", app.ContainerID)
		}
	}

	return nil
}

// CascadeDeleteUser uninstalls all apps for a user, removes user directories,
// and deletes the user record from the database.
func (o *Orchestrator) CascadeDeleteUser(ctx context.Context, userID string) error {
	user, err := o.DB.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Uninstall all user apps.
	apps, err := o.DB.ListAppsByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to list user apps: %w", err)
	}
	for _, app := range apps {
		if err := o.Uninstall(ctx, app.ID, true); err != nil {
			o.Logger.Error("cascade delete: failed to uninstall app",
				logging.F("app_id", app.AppID),
				logging.F("error", err.Error()))
		}
	}

	// Remove user directories.
	userDir := filepath.Join(o.Cfg.Root, "users", user.Username)
	os.RemoveAll(userDir)

	// Delete user record.
	if err := o.DB.DeleteUser(userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	o.Logger.Info("user cascade deleted", logging.F("username", user.Username))
	return nil
}

// generateTokenSecret generates a random 32-byte hex-encoded secret.
func generateTokenSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// RequestPermissions registers a permission request and blocks until the user approves or rejects it,
// or until the context is cancelled, or the request times out (5 minutes).
func (o *Orchestrator) RequestPermissions(ctx context.Context, appID, userID, reason string, scopes []string) (bool, string, error) {
	app, err := o.DB.GetAppByAppIDAndUserID(appID, userID)
	if err != nil {
		return false, "", fmt.Errorf("database query error: %w", err)
	}
	if app == nil {
		return false, "", fmt.Errorf("calling app not found")
	}

	reqID := uuid.NewString()
	req := &PermissionRequest{
		ID:       reqID,
		AppID:    appID,
		AppName:  app.Name,
		Scopes:   scopes,
		Reason:   reason,
		Status:   "pending",
		userID:   userID,
		decision: make(chan bool),
	}

	o.mu.Lock()
	o.pendingPermissions[reqID] = req
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.pendingPermissions, reqID)
		o.mu.Unlock()
	}()

	o.Logger.Info("App requesting permissions dynamically", logging.F("app_id", appID), logging.F("request_id", reqID), logging.F("scopes", scopes))

	// Block until approved, rejected, cancelled, or times out
	select {
	case approved := <-req.decision:
		if !approved {
			return false, "", nil
		}

		// Insert new scopes into DB
		for _, s := range scopes {
			if err := o.DB.InsertAppScope(appID, userID, s); err != nil {
				o.Logger.Error("failed to save new app scope", logging.F("app_id", appID), logging.F("scope", s), logging.F("error", err.Error()))
			}
		}

		// Fetch all scopes from DB to generate new JWT
		allScopes, err := o.DB.ListAppScopes(appID, userID)
		if err != nil {
			return false, "", fmt.Errorf("failed to fetch updated scopes: %w", err)
		}

		// Fetch token secret to sign JWT
		token, err := o.DB.GetAppToken(appID, userID)
		if err != nil || token == nil {
			return false, "", fmt.Errorf("failed to fetch app token secret: %w", err)
		}

		newAppToken, err := o.TokenGenerator([]byte(token.TokenSecret), userID, appID, allScopes)
		if err != nil {
			return false, "", fmt.Errorf("failed to generate new app token: %w", err)
		}

		return true, newAppToken, nil

	case <-ctx.Done():
		return false, "", ctx.Err()

	case <-time.After(5 * time.Minute):
		return false, "", fmt.Errorf("permission request timed out")
	}
}

// ListPendingPermissions returns all pending permission requests for a given user.
func (o *Orchestrator) ListPendingPermissions(userID string) []*PermissionRequest {
	o.mu.RLock()
	defer o.mu.RUnlock()

	list := make([]*PermissionRequest, 0)
	for _, req := range o.pendingPermissions {
		if req.userID == userID {
			list = append(list, req)
		}
	}
	return list
}


// ApprovePermissionRequest approves a pending request by sending true to its channel.
func (o *Orchestrator) ApprovePermissionRequest(reqID string) bool {
	o.mu.RLock()
	req, exists := o.pendingPermissions[reqID]
	o.mu.RUnlock()

	if !exists {
		return false
	}

	select {
	case req.decision <- true:
		req.Status = "approved"
		return true
	default:
		return false
	}
}

// RejectPermissionRequest rejects a pending request by sending false to its channel.
func (o *Orchestrator) RejectPermissionRequest(reqID string) bool {
	o.mu.RLock()
	req, exists := o.pendingPermissions[reqID]
	o.mu.RUnlock()

	if !exists {
		return false
	}

	select {
	case req.decision <- false:
		req.Status = "rejected"
		return true
	default:
		return false
	}
}

