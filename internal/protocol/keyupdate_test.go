package protocol

import (
	"context"
	"testing"

	"github.com/magicbox/core/internal/p2p"
)

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
