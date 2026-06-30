package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminUpgrade_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestAdminUpgrade_ForbiddenForNonAdmin(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create standard user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)

	// Login standard user to get session cookie
	userCookie := getSessionCookieForUser(t, handler, "user", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte("{}")))
	req.AddCookie(userCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestAdminUpgrade_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u2", "admin", string(hash), true)

	// Login admin to get session cookie
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("POST", "/api/v1/admin/upgrade", bytes.NewReader([]byte(`{"image":"magicbox/core:latest"}`)))
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["message"] != "upgrade initiated successfully (mock)" {
		t.Errorf("expected mock upgrade message, got: %s", resp["message"])
	}
}
