package protocol

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// SuccessionCertificate proves that a new identity key (and peer ID) is the legitimate successor
// of a previous one. It is signed by the old identity private key.
type SuccessionCertificate struct {
	OldPeerID    string `json:"old_peer_id"`
	NewPeerID    string `json:"new_peer_id"`
	NewMultiaddr string `json:"new_multiaddr"`
	NewEncPubKey string `json:"new_enc_pub_key"`
	Timestamp    string `json:"timestamp"`
	Signature    string `json:"signature,omitempty"`
}

// SignSuccessionCertificate signs a succession certificate using the old Ed25519 private key.
func SignSuccessionCertificate(oldPriv ed25519.PrivateKey, oldPeerID, newPeerID, newMultiaddr, newEncPubKey string) (*SuccessionCertificate, error) {
	cert := &SuccessionCertificate{
		OldPeerID:    oldPeerID,
		NewPeerID:    newPeerID,
		NewMultiaddr: newMultiaddr,
		NewEncPubKey: newEncPubKey,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	// Marshal fields to sign (excluding signature field)
	data, err := json.Marshal(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate for signing: %w", err)
	}

	sigBytes := ed25519.Sign(oldPriv, data)
	cert.Signature = hex.EncodeToString(sigBytes)
	return cert, nil
}

// VerifySuccessionCertificate verifies that the certificate's signature matches the old peer ID's public key.
func VerifySuccessionCertificate(cert *SuccessionCertificate) error {
	if cert.Signature == "" {
		return fmt.Errorf("missing certificate signature")
	}

	sigBytes, err := hex.DecodeString(cert.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %w", err)
	}

	// Decode old peer ID to get its public key.
	oldPeer, err := peer.Decode(cert.OldPeerID)
	if err != nil {
		return fmt.Errorf("failed to decode old peer ID: %w", err)
	}

	pubKey, err := oldPeer.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("failed to extract public key from old peer ID: %w", err)
	}

	rawPubBytes, err := pubKey.Raw()
	if err != nil {
		return fmt.Errorf("failed to get raw public key bytes: %w", err)
	}

	if len(rawPubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(rawPubBytes))
	}
	oldEdPub := ed25519.PublicKey(rawPubBytes)

	// Create a copy of the cert without the signature to marshal for verification.
	certCopy := *cert
	certCopy.Signature = ""
	data, err := json.Marshal(certCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate for verification: %w", err)
	}

	if !ed25519.Verify(oldEdPub, data, sigBytes) {
		return fmt.Errorf("succession signature verification failed")
	}

	return nil
}

// newKeySuccessionHandler returns a handler for the system:key-succession protocol.
// When a contact rotates their identity key, they broadcast this succession certificate.
// This handler verifies the certificate signature against the contact's old public key,
// and if valid, updates the contact's peer ID, multiaddress, and encryption public key.
func newKeySuccessionHandler(database *db.DB, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		logger.Info("Received identity succession certificate",
			logging.F("from_peer", fromPeerID))

		var cert SuccessionCertificate
		if err := json.Unmarshal(msg.Payload, &cert); err != nil {
			logger.Error("Failed to unmarshal succession certificate",
				logging.F("error", err.Error()))
			return err
		}

		if err := VerifySuccessionCertificate(&cert); err != nil {
			logger.Error("Failed to verify succession certificate",
				logging.F("old_peer", cert.OldPeerID),
				logging.F("new_peer", cert.NewPeerID),
				logging.F("error", err.Error()))
			return err
		}

		// Look up the contact using their old peer ID.
		contact, err := database.GetContactByPeerID(msg.TargetUserID, cert.OldPeerID)
		if err != nil {
			logger.Error("Failed to look up contact for succession update",
				logging.F("old_peer", cert.OldPeerID),
				logging.F("error", err.Error()))
			return err
		}
		if contact == nil {
			logger.Warn("Received succession certificate for unknown contact, ignoring",
				logging.F("old_peer", cert.OldPeerID))
			return nil
		}

		// Update contact details in the database.
		// Replace the old peer ID suffix in their multiaddress if present.
		newMultiaddr := cert.NewMultiaddr
		if strings.Contains(contact.Multiaddr, cert.OldPeerID) {
			newMultiaddr = strings.ReplaceAll(contact.Multiaddr, cert.OldPeerID, cert.NewPeerID)
		}

		if err := database.UpdateContactIdentity(contact.ID, cert.NewPeerID, newMultiaddr, cert.NewEncPubKey); err != nil {
			logger.Error("Failed to update contact identity after succession",
				logging.F("contact_id", contact.ID),
				logging.F("error", err.Error()))
			return err
		}

		logger.Info("Successfully updated contact identity via succession certificate",
			logging.F("contact_id", contact.ID),
			logging.F("display_name", contact.DisplayName),
			logging.F("old_peer_id", cert.OldPeerID),
			logging.F("new_peer_id", cert.NewPeerID))

		return nil
	}
}
