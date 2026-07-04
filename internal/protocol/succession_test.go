package protocol

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/magicbox/core/internal/p2p"
)

func TestSuccessionCertificate_SignAndVerify(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	// We need a valid peer ID derived from the public key.
	_, libp2pPub, err := crypto.KeyPairFromStdKey(&priv)
	if err != nil {
		t.Fatalf("failed to convert standard key to libp2p: %v", err)
	}
	peerID, err := peer.IDFromPublicKey(libp2pPub)
	if err != nil {
		t.Fatalf("failed to generate peer ID: %v", err)
	}

	cert, err := SignSuccessionCertificate(priv, peerID.String(), "new-peer-id", "new-multiaddr", "new-enc-pub")
	if err != nil {
		t.Fatalf("SignSuccessionCertificate failed: %v", err)
	}

	if err := VerifySuccessionCertificate(cert); err != nil {
		t.Fatalf("VerifySuccessionCertificate failed: %v", err)
	}

	// Corrupt signature to verify it fails
	cert.Signature = "aabbcc"
	if err := VerifySuccessionCertificate(cert); err == nil {
		t.Error("expected error for corrupted signature, got nil")
	}
}

func TestKeySuccessionHandler_UpdatesContact(t *testing.T) {
	database, logger := setupTest(t)
	database.CreateUser("user-1", "alice", "hash", false)

	// Add contact with old peer ID.
	_, priv, _ := ed25519.GenerateKey(nil)
	_, libp2pPub, _ := crypto.KeyPairFromStdKey(&priv)
	oldPeerID, _ := peer.IDFromPublicKey(libp2pPub)
	database.AddContact("c-1", "user-1", "Bob", oldPeerID.String(), "/ip4/1.1.1.1/tcp/4001/p2p/"+oldPeerID.String(), "bob-uid", "enc-old")

	// Sign succession certificate.
	newPeerID := "new-peer-123"
	cert, err := SignSuccessionCertificate(priv, oldPeerID.String(), newPeerID, "/ip4/1.1.1.1/tcp/4001/p2p/"+newPeerID, "enc-new")
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
	contact, err := database.GetContactByPeerID("user-1", newPeerID)
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be updated to new peer ID")
	}
	if contact.EncPubKey != "enc-new" {
		t.Errorf("expected enc_pub_key enc-new, got %s", contact.EncPubKey)
	}
	if contact.Multiaddr != "/ip4/1.1.1.1/tcp/4001/p2p/"+newPeerID {
		t.Errorf("expected updated multiaddress, got %s", contact.Multiaddr)
	}

	// Verify old peer ID lookup returns nil.
	oldContact, _ := database.GetContactByPeerID("user-1", oldPeerID.String())
	if oldContact != nil {
		t.Error("expected old contact peer ID to not match anymore")
	}
}
