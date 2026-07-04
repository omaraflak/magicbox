package protocol

import (
	"context"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// newKeyUpdateHandler returns a handler for the system:key-update protocol.
// When a remote peer rotates their encryption key, they broadcast the new
// public key to all contacts. This handler looks up the sender by peer ID
// and updates the stored encryption public key.
func newKeyUpdateHandler(database *db.DB, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		logger.Info("Received encryption key rotation update from peer",
			logging.F("from_peer", fromPeerID))

		newKeyHex := string(msg.Payload)

		contact, err := database.GetContactByPeerID(msg.TargetUserID, fromPeerID)
		if err != nil {
			logger.Error("Failed to look up contact by peer ID for key update",
				logging.F("peer_id", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}
		if contact == nil {
			logger.Warn("Received key update from unknown peer, ignoring",
				logging.F("peer_id", fromPeerID))
			return nil
		}

		if err := database.UpdateContactEncPubKey(contact.ID, newKeyHex); err != nil {
			logger.Error("Failed to update contact encryption key",
				logging.F("contact_id", contact.ID),
				logging.F("error", err.Error()))
			return err
		}

		logger.Info("Successfully updated contact encryption public key",
			logging.F("contact_id", contact.ID),
			logging.F("peer_id", fromPeerID))
		return nil
	}
}
