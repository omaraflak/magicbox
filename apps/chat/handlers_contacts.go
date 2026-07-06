package main

import (
	"encoding/json"
	"net/http"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/sdk"
)

// GET /api/contacts
func handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := env.EnsureScopes([]string{"profile:read", "contacts:read"}, []string{"Read your basic user profile (username, user ID)", "Access contacts to display names and profile invite links"}); err != nil {
		if sdk.WriteConsentError(w, err) {
			return
		}
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get contacts from core: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp.Contacts)
}

// POST /api/contacts/add
func handleAddContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		InviteLink  string `json:"invite_link"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.InviteLink == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "invite_link and display_name are required")
		return
	}

	if err := env.EnsureScopes([]string{"contacts:write"}, []string{"Send contact requests directly to other users"}); err != nil {
		if sdk.WriteConsentError(w, err) {
			return
		}
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.SendContactRequest(ctx, &pb.SendContactRequestRequest{
		InviteLink:  req.InviteLink,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send contact request: "+err.Error())
		return
	}

	if !resp.Success {
		writeError(w, http.StatusBadRequest, resp.StatusMessage)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": resp.StatusMessage,
	})
}
