package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/magicbox/core/internal/logging"
)

// DispatchWebhook sends a webhook payload to a target app's container.
// It looks up the container's internal IP and POSTs to the app's webhook endpoint.
func (o *Orchestrator) DispatchWebhook(ctx context.Context, targetAppID, targetUserID, sourceAppID, sourceUserID string, payload []byte) (int, error) {
	// 1. Look up the target app.
	app, err := o.DB.GetAppByAppIDAndUserID(targetAppID, targetUserID)
	if err != nil {
		return 0, fmt.Errorf("failed to look up target app: %w", err)
	}
	if app == nil {
		return 0, fmt.Errorf("target app %q not found for user %q", targetAppID, targetUserID)
	}
	if app.ContainerID == "" {
		return 0, fmt.Errorf("target app %q has no container", targetAppID)
	}
	if app.Status != "running" {
		return 0, fmt.Errorf("target app %q is not running (status: %s)", targetAppID, app.Status)
	}

	// 2. Inspect container for IP address.
	status, err := o.Docker.InspectContainer(ctx, app.ContainerID)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect target container: %w", err)
	}
	if status.IPAddress == "" {
		return 0, fmt.Errorf("target container has no IP address")
	}

	// 3. Build the webhook URL using stored entry_port and webhook_path.
	webhookPath := app.WebhookPath
	url := fmt.Sprintf("http://%s:%d%s", status.IPAddress, app.EntryPort, webhookPath)

	// 4. Create the HTTP request with a 30s timeout.
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Magicbox-Source-App", sourceAppID)
	req.Header.Set("X-Magicbox-Source-User", sourceUserID)

	// 5. Execute the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	o.Logger.Info("webhook dispatched",
		logging.F("target_app", targetAppID),
		logging.F("source_app", sourceAppID),
		logging.F("status", resp.StatusCode))

	// 6. Return the HTTP status code.
	return resp.StatusCode, nil
}
