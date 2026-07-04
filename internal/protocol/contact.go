package protocol

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// ContactRequestPayload is the data sent in a system:contact-request message.
type ContactRequestPayload struct {
	DisplayName string `json:"display_name"`
	Multiaddr   string `json:"multiaddr"`
	EncPubKey   string `json:"enc_pub_key"`
	UserID      string `json:"user_id"`
}

// ContactAcceptPayload is the data sent in a system:contact-accept message.
type ContactAcceptPayload struct {
	DisplayName string `json:"display_name"`
	Multiaddr   string `json:"multiaddr"`
	EncPubKey   string `json:"enc_pub_key"`
	UserID      string `json:"user_id"`
}

// newContactRequestHandler returns a handler for the system:contact-request protocol.
// When a remote peer sends a contact request, this handler stores it as an
// incoming request in the database for the local user to review and accept/reject.
func newContactRequestHandler(database *db.DB, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		logger.Info("Received contact request from peer",
			logging.F("from_peer", fromPeerID))

		var payload ContactRequestPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			logger.Error("Failed to unmarshal contact request payload",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		id := uuid.NewString()
		if err := database.InsertContactRequest(
			id, msg.TargetUserID, "incoming",
			payload.DisplayName, fromPeerID, payload.Multiaddr,
			payload.UserID, payload.EncPubKey,
		); err != nil {
			logger.Error("Failed to store contact request",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		logger.Info("Stored incoming contact request",
			logging.F("request_id", id),
			logging.F("from_peer", fromPeerID),
			logging.F("display_name", payload.DisplayName))
		return nil
	}
}

// newContactAcceptHandler returns a handler for the system:contact-accept protocol.
// When a remote peer accepts our contact request, this handler creates the contact
// from the stored outgoing request data and removes the request.
func newContactAcceptHandler(database *db.DB, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		logger.Info("Received contact accept from peer",
			logging.F("from_peer", fromPeerID))

		var payload ContactAcceptPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			logger.Error("Failed to unmarshal contact accept payload",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		// Look up the outgoing request we sent to this peer.
		req, err := database.GetContactRequestByPeerID(msg.TargetUserID, fromPeerID)
		if err != nil {
			logger.Error("Failed to look up outgoing request for accept",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}
		if req == nil {
			logger.Warn("Received contact accept but no matching outgoing request found",
				logging.F("from_peer", fromPeerID))
			return nil
		}

		// Create the contact using the stored request data (the accept confirms it).
		contactID := uuid.NewString()
		if err := database.AddContact(
			contactID, req.UserID, req.DisplayName,
			req.PeerID, req.Multiaddr,
			req.TargetUserID, req.EncPubKey,
		); err != nil {
			logger.Error("Failed to create contact from accepted request",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		// Clean up the outgoing request.
		if err := database.DeleteContactRequest(req.UserID, req.ID); err != nil {
			logger.Error("Failed to delete outgoing request after accept",
				logging.F("request_id", req.ID),
				logging.F("error", err.Error()))
		}

		logger.Info("Contact created from accepted request",
			logging.F("contact_id", contactID),
			logging.F("from_peer", fromPeerID),
			logging.F("display_name", req.DisplayName))
		return nil
	}
}
