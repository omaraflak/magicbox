package p2p

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	corecrypto "github.com/magicbox/core/internal/crypto"
)

// encryptOutbound encrypts and signs a message payload for sending to a remote peer.
// It uses the sender's Ed25519 identity key for signing and the recipient's
// X25519 public key (hex-encoded) for encryption.
func encryptOutbound(senderPrivKey crypto.PrivKey, recipientEncPubKeyHex string, msg *Message) (*Message, error) {
	pubBytes, err := hex.DecodeString(recipientEncPubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption public key hex: %w", err)
	}

	curve := ecdh.X25519()
	recipientXPub, err := curve.NewPublicKey(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X25519 public key: %w", err)
	}

	senderEdPriv, err := getEd25519PrivKey(senderPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local signing key: %w", err)
	}

	encryptedPayload, err := corecrypto.EncryptAndSign(senderEdPriv, recipientXPub, msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt/sign payload: %w", err)
	}

	return &Message{
		AppID:        msg.AppID,
		TargetUserID: msg.TargetUserID,
		Payload:      encryptedPayload,
	}, nil
}

// decryptInbound decrypts and verifies an incoming message payload from a remote peer.
// It derives the sender's Ed25519 public key from their peer ID for signature verification
// and uses the local X25519 private key for decryption.
func decryptInbound(recipientEncPriv *ecdh.PrivateKey, senderPeerIDStr string, msg *Message) (*Message, error) {
	senderPeerID, err := peer.Decode(senderPeerIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode sender peer ID: %w", err)
	}

	// Extract the static signing key directly from Peer ID (Public Key Pinning / Cryptographic binding).
	senderPubKey, err := senderPeerID.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to extract sender public key from peer ID: %w", err)
	}

	senderEdPub, err := getEd25519PubKey(senderPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sender public key: %w", err)
	}

	decryptedPayload, err := corecrypto.DecryptAndVerify(recipientEncPriv, senderEdPub, msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt/verify payload: %w", err)
	}

	return &Message{
		AppID:        msg.AppID,
		TargetUserID: msg.TargetUserID,
		Payload:      decryptedPayload,
	}, nil
}

// getEd25519PubKey converts a libp2p public key to a Go standard library Ed25519 public key.
func getEd25519PubKey(pubKey crypto.PubKey) (ed25519.PublicKey, error) {
	rawPubBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw public key: %w", err)
	}
	if len(rawPubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: %d", len(rawPubBytes))
	}
	return ed25519.PublicKey(rawPubBytes), nil
}

// getEd25519PrivKey converts a libp2p private key to a Go standard library Ed25519 private key.
func getEd25519PrivKey(privKey crypto.PrivKey) (ed25519.PrivateKey, error) {
	rawPriv, err := privKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw private key: %w", err)
	}
	// go-libp2p's Ed25519 private key Raw() returns the 64-byte private key (seed + pub)
	if len(rawPriv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid Ed25519 private key size: %d", len(rawPriv))
	}
	return ed25519.PrivateKey(rawPriv), nil
}
