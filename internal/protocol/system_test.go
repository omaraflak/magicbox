package protocol

import (
	"context"
	"testing"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

func setupTest(t *testing.T) (*db.DB, *logging.Logger) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	t.Cleanup(func() { logger.Close() })

	return database, logger
}

func TestKeyUpdateHandler_Success(t *testing.T) {
	database, logger := setupTest(t)

	database.CreateUser("user-1", "alice", "hash", false)
	database.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-id", "old-enc-key-hex")

	handler := newKeyUpdateHandler(database, logger)
	msg := &p2p.Message{
		AppID:        AppIDKeyUpdate,
		TargetUserID: "user-1",
		Payload:      []byte("new-enc-key-hex"),
	}

	err := handler(context.Background(), "peer-123", msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	contact, err := database.GetContactByPeerID("user-1", "peer-123")
	if err != nil {
		t.Fatalf("failed to get contact: %v", err)
	}
	if contact.EncPubKey != "new-enc-key-hex" {
		t.Errorf("expected enc_pub_key %q, got %q", "new-enc-key-hex", contact.EncPubKey)
	}
}

func TestKeyUpdateHandler_UnknownPeerIgnored(t *testing.T) {
	database, logger := setupTest(t)

	database.CreateUser("user-1", "alice", "hash", false)

	handler := newKeyUpdateHandler(database, logger)
	msg := &p2p.Message{
		AppID:        AppIDKeyUpdate,
		TargetUserID: "user-1",
		Payload:      []byte("new-enc-key-hex"),
	}

	err := handler(context.Background(), "unknown-peer", msg)
	if err != nil {
		t.Fatalf("expected no error for unknown peer, got %v", err)
	}
}

func TestRegisterSystemHandlers_DoesNotPanic(t *testing.T) {
	database, logger := setupTest(t)

	// Use a mock service to verify registration works without panic.
	mock := &mockP2PService{handlers: make(map[string]p2p.Handler)}
	RegisterSystemHandlers(mock, database, logger)

	for _, appID := range []string{AppIDKeyUpdate, AppIDContactRequest, AppIDContactAccept} {
		if _, ok := mock.handlers[appID]; !ok {
			t.Errorf("expected %s handler to be registered", appID)
		}
	}
}

// mockP2PService implements p2p.Service for testing handler registration.
type mockP2PService struct {
	handlers map[string]p2p.Handler
}

func (m *mockP2PService) Start(ctx context.Context) error                          { return nil }
func (m *mockP2PService) Stop() error                                              { return nil }
func (m *mockP2PService) HostID() string                                           { return "mock-host" }
func (m *mockP2PService) Multiaddrs() []string                                     { return nil }
func (m *mockP2PService) RegisterHandler(appID string, handler p2p.Handler)        { m.handlers[appID] = handler }
func (m *mockP2PService) SendTo(ctx context.Context, a string, b string, msg *p2p.Message) error {
	return nil
}
