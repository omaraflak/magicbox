package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
)

func setupTestOrchestrator(t *testing.T) (*Orchestrator, *db.DB) {
	tempDir := t.TempDir()
	tempDB := filepath.Join(tempDir, "test.db")

	database, err := db.Open(tempDB)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := &config.Config{
		Root: tempDir,
	}

	orch := NewOrchestrator(database, nil, cfg, logger, nil)
	return orch, database
}

func TestDispatchWebhook_Success(t *testing.T) {
	orch, database := setupTestOrchestrator(t)

	targetAppID := "com.target.app"
	targetUserID := "user-123"
	tokenSecret := "super-secret-webhook-key"
	payload := []byte("hello-webhook")

	// 1. Seed database
	_ = database.CreateUser(targetUserID, "targetuser", "hash", false)
	_ = database.InsertAppToken(targetAppID, targetUserID, tokenSecret)

	// Create test server to receive the webhook
	var receivedSecret string
	var receivedPayload []byte
	var receivedSourceApp string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSecret = r.Header.Get("X-Magicbox-Webhook-Secret")
		receivedSourceApp = r.Header.Get("X-Magicbox-Source-App")
		var err error
		receivedPayload, err = io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Parse test server URL to get port
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	hostParts := strings.Split(u.Host, ":")
	if len(hostParts) != 2 {
		t.Fatalf("unexpected host format: %s", u.Host)
	}
	port, err := strconv.Atoi(hostParts[1])
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	// Insert app configuration pointing to test server port and path
	app := &db.App{
		ID:          "app-target-db-id",
		AppID:       targetAppID,
		UserID:      targetUserID,
		Status:      "running",
		RouteSlug:   "target",
		Image:       "alpine",
		Version:     "1.0.0",
		ContainerID: "container-target",
		EntryPort:   port,
		WebhookPath: "/webhook-endpoint",
	}
	_ = database.InsertApp(app)

	// Dispatch webhook
	statusCode, err := orch.DispatchWebhook(
		context.Background(),
		targetAppID,
		targetUserID,
		"com.source.app",
		"source-user-id",
		"event-type",
		payload,
	)

	if err != nil {
		t.Fatalf("DispatchWebhook failed: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("expected status code 200, got %d", statusCode)
	}

	if receivedSecret != tokenSecret {
		t.Errorf("expected secret header %q, got %q", tokenSecret, receivedSecret)
	}

	if string(receivedPayload) != string(payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(receivedPayload))
	}

	if receivedSourceApp != "com.source.app" {
		t.Errorf("expected source app header 'com.source.app', got %q", receivedSourceApp)
	}
}

func TestDispatchWebhook_MissingToken(t *testing.T) {
	orch, database := setupTestOrchestrator(t)

	targetAppID := "com.target.app"
	targetUserID := "user-123"

	_ = database.CreateUser(targetUserID, "targetuser", "hash", false)
	// Do NOT insert the app token!

	app := &db.App{
		ID:          "app-target-db-id",
		AppID:       targetAppID,
		UserID:      targetUserID,
		Status:      "running",
		RouteSlug:   "target",
		Image:       "alpine",
		Version:     "1.0.0",
		ContainerID: "container-target",
		EntryPort:   8080,
		WebhookPath: "/webhook-endpoint",
	}
	_ = database.InsertApp(app)

	_, err := orch.DispatchWebhook(
		context.Background(),
		targetAppID,
		targetUserID,
		"com.source.app",
		"source-user-id",
		"event-type",
		[]byte("payload"),
	)

	if err == nil {
		t.Fatal("expected error due to missing app token, but got none")
	}

	if !strings.Contains(err.Error(), "app token not found") {
		t.Errorf("expected error message to mention missing app token, got: %v", err)
	}
}
