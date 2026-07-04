package cron

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/keymanager"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/protocol"
)


type MockP2P struct {
	hostID string
}

func (m *MockP2P) Start(ctx context.Context) error { return nil }
func (m *MockP2P) Stop() error                    { return nil }
func (m *MockP2P) HostID() string                 { return m.hostID }
func (m *MockP2P) Multiaddrs() []string           { return []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + m.hostID} }
func (m *MockP2P) RegisterHandler(appID string, handler p2p.Handler) {}
func (m *MockP2P) SendTo(ctx context.Context, peerMultiaddr string, encPubKeyHex string, msg *p2p.Message) error {
	return nil
}

func TestCheckAndRotate(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup Database
	database, err := db.Open(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	if err := database.Migrate(); err != nil {
		t.Fatalf("db migrate: %v", err)
	}

	// Create a user and a contact to verify queueing
	_ = database.CreateUser("user1", "alice", "hash", true)
	_ = database.AddContact("contact1", "user1", "Bob", "remote-peer-id", "/ip4/127.0.0.1/tcp/4001/p2p/remote-peer-id", "bob-uid", "remote-enc-key", "remote-master-key")

	// 2. Setup Config
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)
	paths := keymanager.NewKeyPaths(tempDir)
	keys, err := keymanager.LoadOrGenerate(paths)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}

	mnemonic := keys.Mnemonic
	if mnemonic == "" {
		t.Fatalf("expected mnemonic to be generated")
	}

	cfg := &config.Config{
		Root:          tempDir,
		Keys:          keys,
		MnemonicStore: keymanager.NewMnemonicStore(),
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("logging: %v", err)
	}
	defer logger.Close()

	p2pMock := &MockP2P{hostID: "local-peer-id"}

	// --- TEST 1: Rotation not due yet ---
	checkAndRotate(database, p2pMock, cfg, logger)
	if cfg.Keys.IdentityKeyIndex != 1 || cfg.Keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected indices to remain 1, got id=%d, enc=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}

	// --- TEST 2: Rotation due, but system is LOCKED ---
	pastIdentityTime := time.Now().Add(-16 * 24 * time.Hour)
	pastEncryptionTime := time.Now().Add(-4 * 24 * time.Hour)

	if err := os.Chtimes(paths.IdentityKeyPath, pastIdentityTime, pastIdentityTime); err != nil {
		t.Fatalf("Chtimes identity failed: %v", err)
	}
	if err := os.Chtimes(paths.EncryptionKeyPath, pastEncryptionTime, pastEncryptionTime); err != nil {
		t.Fatalf("Chtimes encryption failed: %v", err)
	}

	checkAndRotate(database, p2pMock, cfg, logger)
	// Indices should not change because system is locked
	if cfg.Keys.IdentityKeyIndex != 1 || cfg.Keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected indices to remain 1 when locked, got id=%d, enc=%d", cfg.Keys.IdentityKeyIndex, cfg.Keys.EncryptionKeyIndex)
	}

	// --- TEST 3: Rotation due, and system is UNLOCKED ---
	cfg.MnemonicStore.Set(mnemonic)

	checkAndRotate(database, p2pMock, cfg, logger)
	// Indices should now be incremented to 2
	if cfg.Keys.IdentityKeyIndex != 2 {
		t.Errorf("expected IdentityKeyIndex to rotate to 2, got %d", cfg.Keys.IdentityKeyIndex)
	}
	if cfg.Keys.EncryptionKeyIndex != 2 {
		t.Errorf("expected EncryptionKeyIndex to rotate to 2, got %d", cfg.Keys.EncryptionKeyIndex)
	}

	// Verify succession certificates and key updates are queued in database message_queue
	msgs, err := database.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 queued messages, got %d", len(msgs))
	}

	hasSuccession := false
	hasUpdate := false
	for _, m := range msgs {
		if m.AppID == protocol.AppIDKeySuccession {
			hasSuccession = true
		}
		if m.AppID == protocol.AppIDKeyUpdate {
			hasUpdate = true
		}
	}

	if !hasSuccession {
		t.Errorf("expected succession certificate message to be enqueued")
	}
	if !hasUpdate {
		t.Errorf("expected encryption key update message to be enqueued")
	}
}

