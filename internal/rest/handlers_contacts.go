package rest

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/invite"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/protocol"
)

func (s *Server) handleListContacts(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	contacts, err := s.db.GetContacts(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load contacts: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, contacts)
}

func (s *Server) handleDeleteContact(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing contact ID")
		return
	}

	if err := s.db.DeleteContact(id, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete contact: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "contact deleted successfully"})
}

func (s *Server) handleGetInvitation(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	peerID := s.p2pService.HostID()
	if peerID == "" {
		writeError(w, http.StatusServiceUnavailable, "P2P network service unavailable")
		return
	}

	// Use relay multiaddress (p2p-circuit) if available to support NAT traversal over the internet.
	// Otherwise, fall back to the first full multiaddress (never a bare peer ID).
	multiaddrs := s.p2pService.Multiaddrs()
	if len(multiaddrs) == 0 {
		writeError(w, http.StatusServiceUnavailable, "P2P network has no listening addresses")
		return
	}

	targetAddr := multiaddrs[0]
	for _, addr := range multiaddrs {
		if strings.Contains(addr, "/p2p-circuit") {
			targetAddr = addr
			break
		}
	}

	// Unmarshal static X25519 public key and encode its raw bytes as hex
	pubKey, err := crypto.UnmarshalX25519PublicKey(s.config.Keys.EncryptionPubPEM)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse local encryption public key: "+err.Error())
		return
	}
	hexPub := hex.EncodeToString(pubKey.Bytes())

	payload := &invite.Payload{
		Multiaddr: targetAddr,
		UserID:    claims.UserID,
		EncPubKey: hexPub,
	}

	inviteLink, err := invite.Build(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build invitation link: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"peer_id":      peerID,
		"invite_link":  inviteLink,
		"multiaddrs":   s.p2pService.Multiaddrs(),
		"invitations":  s.p2pService.Multiaddrs(),
	})
}

// handleSendContactRequest sends a contact request to a remote peer.
// The request is queued for delivery and stored as an outgoing request.
func (s *Server) handleSendContactRequest(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Multiaddr   string `json:"multiaddr"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" || req.Multiaddr == "" {
		writeError(w, http.StatusBadRequest, "display_name and multiaddr (invite link) are required")
		return
	}

	// Parse the invite link to extract the remote peer's info.
	payload, err := invite.Parse(req.Multiaddr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invite link: "+err.Error())
		return
	}

	peerID := invite.ExtractPeerID(payload.Multiaddr)
	if peerID == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: could not extract peer ID")
		return
	}

	if payload.UserID == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: missing user_id")
		return
	}

	// Store as outgoing request.
	reqID := uuid.NewString()
	if err := s.db.InsertContactRequest(
		reqID, claims.UserID, "outgoing", req.DisplayName,
		peerID, payload.Multiaddr, payload.UserID, payload.EncPubKey,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store contact request: "+err.Error())
		return
	}

	// Build the request payload with our own info.
	ourMultiaddrs := s.p2pService.Multiaddrs()
	ourMultiaddr := ""
	if len(ourMultiaddrs) > 0 {
		ourMultiaddr = ourMultiaddrs[0]
		for _, addr := range ourMultiaddrs {
			if strings.Contains(addr, "/p2p-circuit") {
				ourMultiaddr = addr
				break
			}
		}
	}

	pubKey, err := crypto.UnmarshalX25519PublicKey(s.config.Keys.EncryptionPubPEM)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read encryption key")
		return
	}
	ourEncPubHex := hex.EncodeToString(pubKey.Bytes())

	requestPayload, _ := json.Marshal(protocol.ContactRequestPayload{
		DisplayName: claims.Username,
		Multiaddr:   ourMultiaddr,
		EncPubKey:   ourEncPubHex,
		UserID:      claims.UserID,
	})

	// Enqueue the request message for delivery.
	// We create a temporary contact-like struct for EnqueueForContacts.
	tempContact := db.Contact{
		ID:           reqID, // use request ID as contact ID for queue
		Multiaddr:    payload.Multiaddr,
		EncPubKey:    payload.EncPubKey,
		TargetUserID: payload.UserID,
	}
	if err := protocol.EnqueueForContacts(s.db, []db.Contact{tempContact}, protocol.AppIDContactRequest, requestPayload); err != nil {
		s.logger.Error("failed to enqueue contact request", logging.F("error", err.Error()))
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":      reqID,
		"message": "Contact request sent",
	})
}

// handleListContactRequests returns pending incoming contact requests.
func (s *Server) handleListContactRequests(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	direction := r.URL.Query().Get("direction")
	requests, err := s.db.GetContactRequests(claims.UserID, direction)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load contact requests")
		return
	}

	if requests == nil {
		requests = []db.ContactRequest{}
	}
	writeJSON(w, http.StatusOK, requests)
}

// handleAcceptContactRequest accepts an incoming contact request.
// Creates the contact and sends a system:contact-accept message back.
func (s *Server) handleAcceptContactRequest(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing request ID")
		return
	}

	req, err := s.db.GetContactRequest(claims.UserID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if req == nil {
		writeError(w, http.StatusNotFound, "contact request not found")
		return
	}
	if req.Direction != "incoming" {
		writeError(w, http.StatusBadRequest, "can only accept incoming requests")
		return
	}

	// Create the contact from the request data.
	contactID := uuid.NewString()
	if err := s.db.AddContact(
		contactID, claims.UserID, req.DisplayName,
		req.PeerID, req.Multiaddr,
		req.TargetUserID, req.EncPubKey,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create contact: "+err.Error())
		return
	}

	// Send accept message back to the requester.
	pubKey, err := crypto.UnmarshalX25519PublicKey(s.config.Keys.EncryptionPubPEM)
	if err != nil {
		s.logger.Error("failed to read encryption key for accept message", logging.F("error", err.Error()))
	} else {
		ourMultiaddrs := s.p2pService.Multiaddrs()
		ourMultiaddr := ""
		if len(ourMultiaddrs) > 0 {
			ourMultiaddr = ourMultiaddrs[0]
			for _, addr := range ourMultiaddrs {
				if strings.Contains(addr, "/p2p-circuit") {
					ourMultiaddr = addr
					break
				}
			}
		}

		acceptPayload, _ := json.Marshal(protocol.ContactAcceptPayload{
			DisplayName: claims.Username,
			Multiaddr:   ourMultiaddr,
			EncPubKey:   hex.EncodeToString(pubKey.Bytes()),
			UserID:      claims.UserID,
		})

		// Use the newly created contact to enqueue the accept message.
		newContact := db.Contact{
			ID:           contactID,
			Multiaddr:    req.Multiaddr,
			EncPubKey:    req.EncPubKey,
			TargetUserID: req.TargetUserID,
		}
		if err := protocol.EnqueueForContacts(s.db, []db.Contact{newContact}, protocol.AppIDContactAccept, acceptPayload); err != nil {
			s.logger.Error("failed to enqueue contact accept", logging.F("error", err.Error()))
		}
	}

	// Delete the request.
	s.db.DeleteContactRequest(claims.UserID, id)

	writeJSON(w, http.StatusOK, map[string]string{
		"contact_id": contactID,
		"message":    "Contact request accepted",
	})
}

// handleRejectContactRequest rejects and deletes an incoming contact request.
func (s *Server) handleRejectContactRequest(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing request ID")
		return
	}

	if err := s.db.DeleteContactRequest(claims.UserID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete request")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Contact request rejected"})
}

