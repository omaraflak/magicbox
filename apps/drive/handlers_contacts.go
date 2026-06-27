package main

import (
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path

func handleListContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query contacts from core: "+err.Error())
		return
	}

	type ContactJSON struct {
		ID           string `json:"id"`
		DisplayName  string `json:"display_name"`
		Multiaddr    string `json:"multiaddr"`
		TargetUserID string `json:"target_user_id"`
	}

	contacts := []ContactJSON{}
	for _, c := range resp.Contacts {
		contacts = append(contacts, ContactJSON{
			ID:           c.Id,
			DisplayName:  c.DisplayName,
			Multiaddr:    c.Multiaddr,
			TargetUserID: c.TargetUserId,
		})
	}

	writeJSON(w, http.StatusOK, contacts)
}
