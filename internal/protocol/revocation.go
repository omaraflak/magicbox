package protocol

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// MasterRevocationPayload represents the revocation request signature payload.
type MasterRevocationPayload struct {
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

// newMasterRevocationHandler returns a handler for system:master-revocation.
func newMasterRevocationHandler(database *db.DB, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		logger.Info("Received master revocation message",
			logging.F("from_peer", fromPeerID))

		var payload MasterRevocationPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			logger.Error("Failed to unmarshal master revocation payload",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		// Look up the contact in the local database (msg.TargetUserID owns the contact, payload.UserID is the sender's user ID)
		contact, err := database.GetContactByTargetUserID(msg.TargetUserID, payload.UserID)
		if err != nil {
			logger.Error("Failed to look up contact for master revocation",
				logging.F("user_id", msg.TargetUserID),
				logging.F("target_user_id", payload.UserID),
				logging.F("error", err.Error()))
			return err
		}
		if contact == nil {
			logger.Warn("Received master revocation for unknown contact, ignoring",
				logging.F("target_user_id", payload.UserID))
			return nil
		}

		// Decode the contact's stored MasterPubKey
		masterPubBytes, err := hex.DecodeString(contact.MasterPubKey)
		if err != nil {
			logger.Error("Failed to decode contact master public key",
				logging.F("contact_id", contact.ID),
				logging.F("error", err.Error()))
			return err
		}

		if len(masterPubBytes) != ed25519.PublicKeySize {
			logger.Error("Invalid master public key size",
				logging.F("contact_id", contact.ID),
				logging.F("size", len(masterPubBytes)))
			return fmt.Errorf("invalid master public key size")
		}
		masterPubKey := ed25519.PublicKey(masterPubBytes)

		// Verify signature of "REVOKE_MASTER_KEY:" + payload.UserID + ":" + payload.Timestamp
		sigBytes, err := hex.DecodeString(payload.Signature)
		if err != nil {
			logger.Error("Failed to decode revocation signature hex",
				logging.F("contact_id", contact.ID),
				logging.F("error", err.Error()))
			return err
		}

		msgToVerify := []byte("REVOKE_MASTER_KEY:" + payload.UserID + ":" + payload.Timestamp)
		if !ed25519.Verify(masterPubKey, msgToVerify, sigBytes) {
			logger.Error("Revocation signature verification failed",
				logging.F("contact_id", contact.ID))
			return fmt.Errorf("revocation signature verification failed")
		}

		// If valid, update contact status in database to 'revoked'
		if err := database.UpdateContactStatus(contact.ID, "revoked"); err != nil {
			logger.Error("Failed to update contact status to revoked",
				logging.F("contact_id", contact.ID),
				logging.F("error", err.Error()))
			return err
		}

		logger.Info("Successfully revoked master key for contact",
			logging.F("contact_id", contact.ID),
			logging.F("target_user_id", payload.UserID))
		return nil
	}
}
