package protocol

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/p2p"
)

func TestAppCheckHandler_AppInstalled(t *testing.T) {
	database, logger := setupTest(t)

	// Seed Target User and Sender User as contact
	targetUserID := "target-user"
	database.CreateUser(targetUserID, "alice", "hash", false)

	senderUserID := "sender-user"
	database.CreateUser(senderUserID, "bob", "hash", false)

	// Add sender-user as a contact of target-user
	database.AddContact("contact-sender", targetUserID, "Bob", "sender-peer-id", "/ip4/1.2.3.4/tcp/4001/p2p/sender-peer-id", senderUserID, "bob-enc-pub", "bob-master-pub")

	// Install the app for target-user
	err := database.InsertApp(&db.App{
		ID:        "app-drive",
		AppID:     "com.example.drive",
		Name:      "Drive",
		UserID:    targetUserID,
		Status:    "installed",
		RouteSlug: "drive",
		Image:     "drive-image",
	})
	if err != nil {
		t.Fatalf("failed to insert app: %v", err)
	}

	p2pMock := &mockAppCheckP2PService{}
	handler := newAppCheckHandler(database, p2pMock, logger)

	reqPayload := AppCheckRequestPayload{
		RequestID:    "req-123",
		SenderUserID: senderUserID,
		AppID:        "com.example.drive",
	}
	payloadBytes, _ := json.Marshal(reqPayload)

	msg := &p2p.Message{
		AppID:        AppIDAppCheck,
		TargetUserID: targetUserID,
		Payload:      payloadBytes,
	}

	err = handler(context.Background(), "sender-peer-id", msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait briefly for the goroutine sending the response to execute
	time.Sleep(10 * time.Millisecond)

	if p2pMock.sentMsg == nil {
		t.Fatal("expected a response message to be sent")
	}

	if p2pMock.sentMsg.AppID != AppIDAppCheckResponse {
		t.Errorf("expected AppID %q, got %q", AppIDAppCheckResponse, p2pMock.sentMsg.AppID)
	}

	if p2pMock.sentMsg.TargetUserID != senderUserID {
		t.Errorf("expected TargetUserID %q, got %q", senderUserID, p2pMock.sentMsg.TargetUserID)
	}

	var resp AppCheckResponsePayload
	if err := json.Unmarshal(p2pMock.sentMsg.Payload, &resp); err != nil {
		t.Fatalf("failed to unmarshal response payload: %v", err)
	}

	if resp.RequestID != "req-123" {
		t.Errorf("expected request ID %q, got %q", "req-123", resp.RequestID)
	}

	if resp.AppID != "com.example.drive" {
		t.Errorf("expected app ID %q, got %q", "com.example.drive", resp.AppID)
	}

	if !resp.Installed {
		t.Error("expected installed to be true")
	}
}

func TestAppCheckHandler_AppNotInstalled(t *testing.T) {
	database, logger := setupTest(t)

	// Seed Target User and Sender User as contact
	targetUserID := "target-user"
	database.CreateUser(targetUserID, "alice", "hash", false)

	senderUserID := "sender-user"
	database.CreateUser(senderUserID, "bob", "hash", false)

	database.AddContact("contact-sender", targetUserID, "Bob", "sender-peer-id", "/ip4/1.2.3.4/tcp/4001/p2p/sender-peer-id", senderUserID, "bob-enc-pub", "bob-master-pub")

	p2pMock := &mockAppCheckP2PService{}
	handler := newAppCheckHandler(database, p2pMock, logger)

	reqPayload := AppCheckRequestPayload{
		RequestID:    "req-456",
		SenderUserID: senderUserID,
		AppID:        "com.example.nonexistent",
	}
	payloadBytes, _ := json.Marshal(reqPayload)

	msg := &p2p.Message{
		AppID:        AppIDAppCheck,
		TargetUserID: targetUserID,
		Payload:      payloadBytes,
	}

	err := handler(context.Background(), "sender-peer-id", msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if p2pMock.sentMsg == nil {
		t.Fatal("expected a response message to be sent")
	}

	var resp AppCheckResponsePayload
	if err := json.Unmarshal(p2pMock.sentMsg.Payload, &resp); err != nil {
		t.Fatalf("failed to unmarshal response payload: %v", err)
	}

	if resp.Installed {
		t.Error("expected installed to be false")
	}
}

func TestAppCheckResponseHandler(t *testing.T) {
	_, logger := setupTest(t)

	requestID := "req-789"
	ch := RegisterAppCheckRequest(requestID)
	defer DeregisterAppCheckRequest(requestID)

	handler := newAppCheckResponseHandler(logger)

	respPayload := AppCheckResponsePayload{
		RequestID: requestID,
		AppID:     "com.example.chat",
		Installed: true,
	}
	payloadBytes, _ := json.Marshal(respPayload)

	msg := &p2p.Message{
		AppID:        AppIDAppCheckResponse,
		TargetUserID: "sender-user",
		Payload:      payloadBytes,
	}

	err := handler(context.Background(), "target-peer-id", msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case installed := <-ch:
		if !installed {
			t.Error("expected installed to resolve as true")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for ResolveAppCheckRequest")
	}
}

// mockAppCheckP2PService implements p2p.Service for testing.
type mockAppCheckP2PService struct {
	sentMsg *p2p.Message
}

func (m *mockAppCheckP2PService) Start(ctx context.Context) error { return nil }
func (m *mockAppCheckP2PService) Stop() error                     { return nil }
func (m *mockAppCheckP2PService) HostID() string                  { return "mock-host" }
func (m *mockAppCheckP2PService) Multiaddrs() []string            { return nil }
func (m *mockAppCheckP2PService) RegisterHandler(appID string, handler p2p.Handler) {}
func (m *mockAppCheckP2PService) SendTo(ctx context.Context, peerMultiaddr string, encPubKeyHex string, msg *p2p.Message) error {
	m.sentMsg = msg
	return nil
}
