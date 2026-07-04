package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminCreateUser_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user to get cookie
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("admin-id", "admin", string(hash), true)
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	body, _ := json.Marshal(map[string]interface{}{
		"username": "newuser",
		"password": "password123",
		"is_admin": false,
	})

	req := httptest.NewRequest("POST", "/api/v1/admin/users", bytes.NewReader(body))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	u, err := database.GetUserByUsername("newuser")
	if err != nil || u == nil {
		t.Fatalf("expected newuser to exist in database, got error/nil: %v", err)
	}
}
