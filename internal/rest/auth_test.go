package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionTokens(t *testing.T) {
	secret := []byte("my-super-secret-key-1234567890123")
	userID := "user-123"
	username := "omar"
	isAdmin := true

	token, err := GenerateSessionToken(secret, userID, username, isAdmin)
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}

	claims, err := ValidateSessionToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateSessionToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %q, got %q", userID, claims.UserID)
	}

	if claims.Username != username {
		t.Errorf("expected Username %q, got %q", username, claims.Username)
	}

	if claims.IsAdmin != isAdmin {
		t.Errorf("expected IsAdmin %t, got %t", isAdmin, claims.IsAdmin)
	}
}

func TestCSRFTokens(t *testing.T) {
	token, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken failed: %v", err)
	}

	if len(token) != 64 { // 32 bytes in hex = 64 characters
		t.Errorf("expected hex length 64, got %d", len(token))
	}

	if !ValidateCSRFToken(token, token) {
		t.Error("expected token to match itself")
	}

	if ValidateCSRFToken(token, "invalid-token") {
		t.Error("expected validation to fail for mismatched token")
	}
}

func TestAppAccessMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := AppAccessMiddleware()(handler)

	// Case 1: missing claims
	req := httptest.NewRequest("GET", "/u/omar/drive/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	// Case 2: authorized access (username matches claims)
	claims := &SessionClaims{
		UserID:   "user-123",
		Username: "omar",
	}
	ctx := context.WithValue(req.Context(), contextKeyUser, claims)
	req = req.WithContext(ctx)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Case 3: unauthorized access (username mismatch)
	req2 := httptest.NewRequest("GET", "/u/test/drive/", nil)
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()
	middleware.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr2.Code)
	}
}
