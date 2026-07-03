package rest

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// MockP2PService implements p2p.Service for testing.
type MockP2PService struct {
	hostID string
}

func (m *MockP2PService) Start(ctx context.Context) error { return nil }
func (m *MockP2PService) Stop() error                    { return nil }
func (m *MockP2PService) HostID() string                 { return m.hostID }
func (m *MockP2PService) Multiaddrs() []string           { return []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + m.hostID} }
func (m *MockP2PService) RegisterHandler(appID string, handler p2p.Handler) {}
func (m *MockP2PService) SendTo(ctx context.Context, dest string, msg *p2p.Message) error {
	return nil
}

// setupTestServer initializes a fresh test server instance.
func setupTestServer(t *testing.T) (http.Handler, *db.DB, *config.Config) {
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

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}
	privPEM, _ := crypto.MarshalPrivateKey(edPriv)
	pubPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	encKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)
	encPubPEM, _ := crypto.MarshalPublicKey(xPriv.PublicKey())

	cfg := &config.Config{
		Root:             tempDir,
		JWTSecret:        []byte("my-test-super-secret-key-signature-123"),
		PrivateKeyPEM:    privPEM,
		PublicKeyPEM:     pubPEM,
		EncryptionKeyPEM: encKeyPEM,
		EncryptionPubPEM: encPubPEM,
		IdentityKeyIndex:   0,
		EncryptionKeyIndex: 0,
	}

	p2pMock := &MockP2PService{hostID: "QmbQGs4z4UYae7oBDmhyBbyEg6bh9LGQLqDBeVY3GY8x5H"}
	orch := core.NewOrchestrator(database, nil, cfg, logger, GenerateAppToken)

	server := NewServer(cfg, database, nil, logger, orch, p2pMock)
	server.onRestart = func() {}
	return server.Handler(), database, cfg
}
