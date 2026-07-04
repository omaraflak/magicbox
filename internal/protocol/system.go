// Package protocol implements Magicbox system-level P2P message handlers.
// Each system:* protocol (key rotation, succession, contact requests, etc.)
// is registered as a named handler on the P2P service, keeping main.go clean
// and making it trivial to add new system protocols.
package protocol

import (
	"context"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// System protocol AppIDs for P2P messages.
const (
	AppIDKeyUpdate      = "system:key-update"
	AppIDKeySuccession  = "system:key-succession"
	AppIDContactRequest = "system:contact-request"
	AppIDContactAccept  = "system:contact-accept"
)

// RegisterSystemHandlers registers all system:* P2P message handlers on the given service.
// Each handler is registered by its AppID so the P2P layer dispatches directly
// without a manual switch/if-else chain.
func RegisterSystemHandlers(service p2p.Service, database *db.DB, logger *logging.Logger) {
	service.RegisterHandler(AppIDKeyUpdate, newKeyUpdateHandler(database, logger))

	// Future system handlers:
	// service.RegisterHandler("system:key-succession", newKeySuccessionHandler(...))
	// service.RegisterHandler("system:contact-request", newContactRequestHandler(...))
	// service.RegisterHandler("system:contact-accept", newContactAcceptHandler(...))
}

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
