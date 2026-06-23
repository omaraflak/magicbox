// Package rest provides HTTP handlers and middleware for the Magicbox REST API.
package rest

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// SessionCookieName is the cookie name for user sessions.
	// The __Host- prefix enforces Secure, Path=/, and no Domain attribute.
	SessionCookieName = "__Host-magicbox_session"

	// CSRFCookieName is the cookie name for CSRF tokens.
	CSRFCookieName = "__Host-magicbox_csrf"

	// SessionDuration is the lifetime of a session token.
	SessionDuration = 24 * time.Hour
)

// SessionClaims are the JWT claims for a user session.
type SessionClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	IsAdmin  bool   `json:"adm"`
}

// AppTokenClaims are the JWT claims for per-app authentication tokens.
type AppTokenClaims struct {
	jwt.RegisteredClaims
	UserID string   `json:"uid"`
	AppID  string   `json:"app"`
	Scopes []string `json:"scp"`
}

// GenerateSessionToken creates a signed JWT for a user session.
func GenerateSessionToken(secret []byte, userID, username string, isAdmin bool) (string, error) {
	now := time.Now()
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateSessionToken parses and validates a session JWT.
// It rejects any signing method other than HMAC (including "none").
func ValidateSessionToken(secret []byte, tokenStr string) (*SessionClaims, error) {
	claims := &SessionClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// Reject any algorithm that is not HMAC — this explicitly blocks the "none" algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid session token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid session token")
	}
	return claims, nil
}

// GenerateAppToken creates a signed JWT for per-app authentication.
func GenerateAppToken(secret []byte, userID, appID string, scopes []string) (string, error) {
	now := time.Now()
	claims := AppTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		AppID:  appID,
		Scopes: scopes,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateAppToken parses and validates an app JWT.
// It rejects any signing method other than HMAC (including "none").
func ValidateAppToken(secret []byte, tokenStr string) (*AppTokenClaims, error) {
	claims := &AppTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid app token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid app token")
	}
	return claims, nil
}

// ParseAppTokenUnverified parses an app JWT without validating its signature.
func ParseAppTokenUnverified(tokenStr string) (*AppTokenClaims, error) {
	claims := &AppTokenClaims{}
	parser := jwt.NewParser()
	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unverified token: %w", err)
	}
	return claims, nil
}

// SetSessionCookie writes a secure session cookie to the response.
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(SessionDuration.Seconds()),
	})
}

// ClearSessionCookie removes the session cookie by setting MaxAge to -1.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// GetSessionCookie reads the session token from the request cookie.
func GetSessionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", fmt.Errorf("session cookie not found: %w", err)
	}
	return cookie.Value, nil
}

// GenerateCSRFToken generates a cryptographically random CSRF token (32 bytes, hex-encoded).
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ValidateCSRFToken compares two CSRF tokens using constant-time comparison
// to prevent timing attacks.
func ValidateCSRFToken(expected, actual string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
