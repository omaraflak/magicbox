package protocol

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/magicbox/core/internal/p2p"
)

func TestMasterRevocationHandler_Success(t *testing.T) {
	database, logger := setupTest(t)

	// Create user Bob
	bobUserID := "bob-user"
	database.CreateUser(bobUserID, "bob", "hash", false)

	// Generate Alice's master key-pair.
	alicePub, alicePriv, _ := ed25519.GenerateKey(nil)
	alicePubHex := hex.EncodeToString(alicePub)
	aliceUserID := "alice-user"

	// Add Alice as Bob's contact.
	database.AddContact("alice-contact", bobUserID, "Alice", "alice-peer-id", "/ip4/1.1.1.1/p2p/alice-peer-id", aliceUserID, "alice-enc-pub", alicePubHex)

	// Verify Bob's contact for Alice starts as 'active'
	cBefore, err := database.GetContactByID("alice-contact", bobUserID)
	if err != nil {
		t.Fatalf("failed to query contact: %v", err)
	}
	if cBefore.Status != "active" {
		t.Fatalf("expected initial status to be active, got %q", cBefore.Status)
	}

	// Create Alice's revocation payload
	timestamp := "2026-07-04T12:00:00Z"
	msgToSign := []byte("REVOKE_MASTER_KEY:" + aliceUserID + ":" + timestamp)
	sigBytes := ed25519.Sign(alicePriv, msgToSign)
	sigHex := hex.EncodeToString(sigBytes)

	payloadBytes, err := json.Marshal(MasterRevocationPayload{
		UserID:    aliceUserID,
		Timestamp: timestamp,
		Signature: sigHex,
	})
	if err != nil {
		t.Fatalf("failed to marshal revocation payload: %v", err)
	}

	msg := &p2p.Message{
		AppID:        AppIDMasterRevocation,
		TargetUserID: bobUserID,
		Payload:      payloadBytes,
	}

	handler := newMasterRevocationHandler(database, logger)
	if err := handler(context.Background(), "alice-peer-id", msg); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	// Verify contact status is updated to 'revoked'
	cAfter, err := database.GetContactByID("alice-contact", bobUserID)
	if err != nil {
		t.Fatalf("failed to query contact: %v", err)
	}
	if cAfter.Status != "revoked" {
		t.Errorf("expected contact status 'revoked', got %q", cAfter.Status)
	}
}

func TestMasterRevocationHandler_InvalidSignature(t *testing.T) {
	database, logger := setupTest(t)

	// Create user Bob
	bobUserID := "bob-user"
	database.CreateUser(bobUserID, "bob", "hash", false)

	// Generate Alice's master key-pair.
	alicePub, _, _ := ed25519.GenerateKey(nil)
	alicePubHex := hex.EncodeToString(alicePub)
	aliceUserID := "alice-user"

	// Add Alice as Bob's contact.
	database.AddContact("alice-contact", bobUserID, "Alice", "alice-peer-id", "/ip4/1.1.1.1/p2p/alice-peer-id", aliceUserID, "alice-enc-pub", alicePubHex)

	// Invalid signature hex
	payloadBytes, _ := json.Marshal(MasterRevocationPayload{
		UserID:    aliceUserID,
		Timestamp: "2026-07-04T12:00:00Z",
		Signature: "deadbeef",
	})

	msg := &p2p.Message{
		AppID:        AppIDMasterRevocation,
		TargetUserID: bobUserID,
		Payload:      payloadBytes,
	}

	handler := newMasterRevocationHandler(database, logger)
	if err := handler(context.Background(), "alice-peer-id", msg); err == nil {
		t.Error("expected error for invalid signature, got nil")
	}

	// Verify contact status is still 'active'
	cAfter, _ := database.GetContactByID("alice-contact", bobUserID)
	if cAfter.Status != "active" {
		t.Errorf("expected contact status to remain 'active', got %q", cAfter.Status)
	}
}
