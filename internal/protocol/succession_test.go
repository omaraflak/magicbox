package protocol

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/magicbox/core/internal/p2p"
)

func TestSuccessionCertificate_SignAndVerify(t *testing.T) {
	masterPub, masterPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	masterPubKeyHex := hex.EncodeToString(masterPub)

	cert, err := SignSuccessionCertificate(masterPriv, masterPubKeyHex, "old-peer-id", "new-peer-id", "new-multiaddr", "new-enc-pub")
	if err != nil {
		t.Fatalf("SignSuccessionCertificate failed: %v", err)
	}

	if err := VerifySuccessionCertificate(cert, masterPub); err != nil {
		t.Fatalf("VerifySuccessionCertificate failed: %v", err)
	}

	// Corrupt signature to verify it fails
	cert.Signature = "aabbcc"
	if err := VerifySuccessionCertificate(cert, masterPub); err == nil {
		t.Error("expected error for corrupted signature, got nil")
	}
}

func TestKeySuccessionHandler_UpdatesContact(t *testing.T) {
	database, logger := setupTest(t)
	database.CreateUser("user-1", "alice", "hash", false)

	// Generate master key-pair.
	masterPub, masterPriv, _ := ed25519.GenerateKey(nil)
	masterPubKeyHex := hex.EncodeToString(masterPub)

	// Add contact with old peer ID.
	_, priv, _ := ed25519.GenerateKey(nil)
	_, libp2pPub, _ := crypto.KeyPairFromStdKey(&priv)
	oldPeerID, _ := peer.IDFromPublicKey(libp2pPub)
	database.AddContact("c-1", "user-1", "Bob", oldPeerID.String(), "/ip4/1.1.1.1/tcp/4001/p2p/"+oldPeerID.String(), "bob-uid", "enc-old", masterPubKeyHex)

	// Sign succession certificate.
	newPeerID := "new-peer-123"
	cert, err := SignSuccessionCertificate(masterPriv, masterPubKeyHex, oldPeerID.String(), newPeerID, "/ip4/1.1.1.1/tcp/4001/p2p/"+newPeerID, "enc-new")
	if err != nil {
		t.Fatalf("failed to sign cert: %v", err)
	}

	handler := newKeySuccessionHandler(database, logger)
	payload, _ := json.Marshal(cert)
	msg := &p2p.Message{
		AppID:        AppIDKeySuccession,
		TargetUserID: "user-1",
		Payload:      payload,
	}

	if err := handler(context.Background(), oldPeerID.String(), msg); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	// Verify contact was updated.
	contacts, err := database.GetContactsByPeerID("user-1", newPeerID)
	if err != nil {
		t.Fatalf("GetContactsByPeerID failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact with new peer ID, got %d", len(contacts))
	}
	if contacts[0].EncPubKey != "enc-new" {
		t.Errorf("expected enc_pub_key enc-new, got %s", contacts[0].EncPubKey)
	}
	if contacts[0].Multiaddr != "/ip4/1.1.1.1/tcp/4001/p2p/"+newPeerID {
		t.Errorf("expected updated multiaddress, got %s", contacts[0].Multiaddr)
	}

	// Verify old peer ID lookup returns empty.
	oldContacts, _ := database.GetContactsByPeerID("user-1", oldPeerID.String())
	if len(oldContacts) != 0 {
		t.Error("expected old contact peer ID to not match anymore")
	}
}
