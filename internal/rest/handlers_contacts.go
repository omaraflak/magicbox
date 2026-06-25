package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
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

	targetUserID := ""
	if u, parseErr := url.Parse(req.Multiaddr); parseErr == nil {
		targetUserID = u.Query().Get("user_id")
	}
	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid multiaddr: missing user_id query parameter")
		return
	}

	id := uuid.NewString()
	if err := s.db.AddContact(id, claims.UserID, req.DisplayName, req.Multiaddr, targetUserID); err != nil {
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

	// Use relay multiaddress (p2p-circuit) if available to support NAT traversal over the internet
	targetAddr := peerID
	for _, addr := range s.p2pService.Multiaddrs() {
		if strings.Contains(addr, "/p2p-circuit") {
			targetAddr = addr
			break
		}
	}

	inviteLink := fmt.Sprintf("magicbox://invite/%s?user_id=%s", targetAddr, claims.UserID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"peer_id":      peerID,
		"invite_link":  inviteLink,
		"multiaddrs":   s.p2pService.Multiaddrs(),
		"invitations":  s.p2pService.Multiaddrs(),
	})
}
