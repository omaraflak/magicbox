package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminCreateRegistry_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user to get cookie
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("admin-id", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	body, _ := json.Marshal(map[string]interface{}{
		"prefix": "docker.io/myorg",
	})

	req := httptest.NewRequest("POST", "/api/v1/admin/registries", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	registries, err := database.ListRegistries()
	if err != nil {
		t.Fatalf("failed to list registries: %v", err)
	}

	found := false
	for _, reg := range registries {
		if reg.Prefix == "docker.io/myorg" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find registry prefix 'docker.io/myorg' in database registries: %+v", registries)
	}
}
