package rest

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/magicbox/core/internal/crypto"
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

// InvitationPayload represents the fields encoded in the invitation link.
type InvitationPayload struct {
	Multiaddr string `json:"multiaddr"`
	UserID    string `json:"user_id"`
	EncPubKey string `json:"enc_pub_key"`
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

	if !strings.HasPrefix(req.Multiaddr, "magicbox://invite/") {
		writeError(w, http.StatusBadRequest, "invalid invite link: must start with magicbox://invite/")
		return
	}

	b64Payload := strings.TrimPrefix(req.Multiaddr, "magicbox://invite/")
	if b64Payload == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: missing payload")
		return
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(b64Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid base64 payload: "+err.Error())
		return
	}

	var payload InvitationPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload JSON: "+err.Error())
		return
	}

	targetUserID := payload.UserID
	encPubKey := payload.EncPubKey

	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid invite link: missing user_id")
		return
	}

	id := uuid.NewString()
	if err := s.db.AddContact(id, claims.UserID, req.DisplayName, req.Multiaddr, targetUserID, encPubKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save contact: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":             id,
		"display_name":   req.DisplayName,
		"multiaddr":      req.Multiaddr,
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
	pubKey, err := crypto.UnmarshalX25519PublicKey(s.config.EncryptionPubPEM)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse local encryption public key: "+err.Error())
		return
	}
	hexPub := hex.EncodeToString(pubKey.Bytes())

	payload := InvitationPayload{
		Multiaddr: targetAddr,
		UserID:    claims.UserID,
		EncPubKey: hexPub,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal invitation payload: "+err.Error())
		return
	}

	b64Payload := base64.URLEncoding.EncodeToString(payloadBytes)
	inviteLink := fmt.Sprintf("magicbox://invite/%s", b64Payload)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"peer_id":      peerID,
		"invite_link":  inviteLink,
		"multiaddrs":   s.p2pService.Multiaddrs(),
		"invitations":  s.p2pService.Multiaddrs(),
	})
}
