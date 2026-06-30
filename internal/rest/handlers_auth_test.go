package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupFlow_Success(t *testing.T) {
	handler, _, cfg := setupTestServer(t)

	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected setup status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Capture the session cookie
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set on setup auto-login")
	}

	// Verify JWT claims inside the cookie
	claims, err := ValidateSessionToken(cfg.JWTSecret, sessionCookie.Value)
	if err != nil {
		t.Fatalf("failed to validate setup session token: %v", err)
	}
	if claims.Username != "omar" {
		t.Errorf("expected Username 'omar' in cookie, got %q", claims.Username)
	}
}

func TestSetupFlow_SecondTimeForbidden(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)

	// Perform setup successfully once
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected initial setup status 201, got %d", rr.Code)
	}

	// Try to perform setup again
	req = httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403 when setup is run again, got %d", rr.Code)
	}
}

func TestLoginFlow_WrongPassword(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	// Create user first via setup
	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pre-setup failed: %d", rr.Code)
	}

	// Login with invalid credentials
	loginBody := map[string]string{
		"username": "omar",
		"password": "wrongpassword",
	}
	loginBytes, _ := json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 on wrong password, got %d", rr.Code)
	}
}

func TestLoginFlow_Success(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	// Create user first via setup
	setupBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	bodyBytes, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/setup", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pre-setup failed: %d", rr.Code)
	}

	// Login with valid credentials
	loginBody := map[string]string{
		"username": "omar",
		"password": "mypassword123",
	}
	loginBytes, _ := json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBytes))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 on correct login, got %d", rr.Code)
	}
}

func TestLogout(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	cookies := rr.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			clearedCookie = c
			break
		}
	}
	if clearedCookie == nil || clearedCookie.MaxAge != -1 {
		t.Errorf("expected cookie to be cleared (MaxAge=-1), got %+v", clearedCookie)
	}
}

func TestMeEndpoint_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 when unauthenticated, got %d", rr.Code)
	}
}

func TestMeEndpoint_Success(t *testing.T) {
	handler, database, cfg := setupTestServer(t)

	// Create user in DB to satisfy database lookup
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)

	// Authenticated request to /me
	token, _ := GenerateSessionToken(cfg.JWTSecret, userID, "omar", false)
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
		Path:  "/",
	})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["username"] != "omar" {
		t.Errorf("expected username 'omar', got %v", resp["username"])
	}
}

func getSessionCookieForUser(t *testing.T, handler http.Handler, username, password string) *http.Cookie {
	loginBody := map[string]string{
		"username": username,
		"password": password,
	}
	bodyBytes, _ := json.Marshal(loginBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("login failed for %s: %d", username, rr.Code)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			return c
		}
	}
	t.Fatalf("session cookie not found for %s", username)
	return nil
}
