package p2p

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	corecrypto "github.com/magicbox/core/internal/crypto"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Generate sender identity key
	senderMnemonic, err := corecrypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate sender mnemonic: %v", err)
	}
	senderEdPriv, err := corecrypto.DeriveIdentityKey(senderMnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive sender identity key: %v", err)
	}

	// Wrap sender's Ed25519 key as a libp2p private key
	senderLibp2pPriv, _, err := crypto.KeyPairFromStdKey(&senderEdPriv)
	if err != nil {
		t.Fatalf("failed to create libp2p key pair: %v", err)
	}

	// Generate receiver encryption key
	receiverMnemonic, err := corecrypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate receiver mnemonic: %v", err)
	}
	receiverXPriv, err := corecrypto.DeriveEncryptionKey(receiverMnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive receiver encryption key: %v", err)
	}
	receiverEncPubHex := hex.EncodeToString(receiverXPriv.PublicKey().Bytes())

	// Derive sender's peer ID (needed for decryptInbound)
	senderPeerID, err := peer.IDFromPrivateKey(senderLibp2pPriv)
	if err != nil {
		t.Fatalf("failed to derive sender peer ID: %v", err)
	}

	// Original message
	original := &Message{
		AppID:        "test-app",
		TargetUserID: "user-1",
		Payload:      []byte("hello world"),
	}

	// Encrypt
	encrypted, err := encryptOutbound(senderLibp2pPriv, receiverEncPubHex, original)
	if err != nil {
		t.Fatalf("encryptOutbound failed: %v", err)
	}

	if encrypted.AppID != original.AppID {
		t.Errorf("expected AppID %q, got %q", original.AppID, encrypted.AppID)
	}
	if string(encrypted.Payload) == string(original.Payload) {
		t.Error("encrypted payload should differ from original")
	}

	// Decrypt
	decrypted, err := decryptInbound(receiverXPriv, senderPeerID.String(), encrypted)
	if err != nil {
		t.Fatalf("decryptInbound failed: %v", err)
	}

	if decrypted.AppID != original.AppID {
		t.Errorf("expected AppID %q, got %q", original.AppID, decrypted.AppID)
	}
	if string(decrypted.Payload) != string(original.Payload) {
		t.Errorf("expected payload %q, got %q", original.Payload, decrypted.Payload)
	}
}

func TestEncryptOutbound_InvalidHexFails(t *testing.T) {
	mnemonic, _ := corecrypto.GenerateMnemonic()
	edPriv, _ := corecrypto.DeriveIdentityKey(mnemonic, 0)
	libp2pPriv, _, _ := crypto.KeyPairFromStdKey(&edPriv)

	msg := &Message{AppID: "test", Payload: []byte("data")}

	_, err := encryptOutbound(libp2pPriv, "not-valid-hex!!!", msg)
	if err == nil {
		t.Error("expected error for invalid hex, got nil")
	}
}

func TestDecryptInbound_InvalidPeerIDFails(t *testing.T) {
	mnemonic, _ := corecrypto.GenerateMnemonic()
	xPriv, _ := corecrypto.DeriveEncryptionKey(mnemonic, 0)

	msg := &Message{AppID: "test", Payload: []byte("data")}

	_, err := decryptInbound(xPriv, "not-a-valid-peer-id", msg)
	if err == nil {
		t.Error("expected error for invalid peer ID, got nil")
	}
}

func TestGetEd25519PubKey_ValidKey(t *testing.T) {
	mnemonic, _ := corecrypto.GenerateMnemonic()
	edPriv, _ := corecrypto.DeriveIdentityKey(mnemonic, 0)
	libp2pPriv, _, _ := crypto.KeyPairFromStdKey(&edPriv)

	pubKey := libp2pPriv.GetPublic()
	edPub, err := getEd25519PubKey(pubKey)
	if err != nil {
		t.Fatalf("getEd25519PubKey failed: %v", err)
	}
	if len(edPub) != ed25519.PublicKeySize {
		t.Errorf("expected public key size %d, got %d", ed25519.PublicKeySize, len(edPub))
	}
}

func TestGetEd25519PrivKey_ValidKey(t *testing.T) {
	mnemonic, _ := corecrypto.GenerateMnemonic()
	edPriv, _ := corecrypto.DeriveIdentityKey(mnemonic, 0)
	libp2pPriv, _, _ := crypto.KeyPairFromStdKey(&edPriv)

	stdPriv, err := getEd25519PrivKey(libp2pPriv)
	if err != nil {
		t.Fatalf("getEd25519PrivKey failed: %v", err)
	}
	if len(stdPriv) != ed25519.PrivateKeySize {
		t.Errorf("expected private key size %d, got %d", ed25519.PrivateKeySize, len(stdPriv))
	}
}
