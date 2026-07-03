package rest

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/invite"
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

func (s *Server) handleCreateContact(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "display_name and multiaddr are required")
		return
	}

	payload, err := invite.Parse(req.Multiaddr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invite link: "+err.Error())
		return
	}

	peerID := invite.ExtractPeerID(payload.Multiaddr)
	if peerID == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: could not extract peer ID from multiaddr")
		return
	}

	targetUserID := payload.UserID
	encPubKey := payload.EncPubKey

	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: missing user_id")
		return
	}

	id := uuid.NewString()
	if err := s.db.AddContact(id, claims.UserID, req.DisplayName, peerID, payload.Multiaddr, targetUserID, encPubKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save contact: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":             id,
		"display_name":   req.DisplayName,
		"peer_id":        peerID,
		"multiaddr":      payload.Multiaddr,
		"target_user_id": targetUserID,
	})
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
