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
