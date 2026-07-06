package main

import (
	"net/http"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/sdk"
)

// GET /api/profile
func handleProfile(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get profile from core: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":    resp.UserId,
		"username":   resp.Username,
		"created_at": resp.CreatedAt,
	})
}
