package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey string

const contextKeyUser contextKey = "user"

// AuthMiddleware validates the session JWT from the cookie and injects
// the claims into the request context. Returns 401 on any failure.
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := GetSessionCookie(r)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			claims, err := ValidateSessionToken(secret, tokenStr)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired session")
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AppAccessMiddleware enforces that the logged-in user can only access their own apps.
// Must be chained after AuthMiddleware.
func AppAccessMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r)
			if claims == nil {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			// Path format: /u/{username}/{app_id}/...
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 && parts[1] == "u" {
				targetUsername := parts[2]
				if claims.Username != targetUsername {
					writeJSONError(w, http.StatusForbidden, "access denied to this app instance")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AdminMiddleware checks that the authenticated user has admin privileges.
// Must be used after AuthMiddleware. Returns 403 if not an admin.
func AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r)
			if claims == nil || !claims.IsAdmin {
				writeJSONError(w, http.StatusForbidden, "admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts session claims from the request context.
// Returns nil if no claims are present.
func GetUserFromContext(r *http.Request) *SessionClaims {
	claims, _ := r.Context().Value(contextKeyUser).(*SessionClaims)
	return claims
}

// SecurityHeadersMiddleware sets security-related HTTP headers on every response.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// CSRFMiddleware enforces CSRF token validation on state-changing requests
// (POST, PUT, DELETE, PATCH) using the double-submit cookie pattern.
// GET, HEAD, and OPTIONS requests pass through without validation.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" {
				writeJSONError(w, http.StatusForbidden, "missing CSRF token")
				return
			}

			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil {
				writeJSONError(w, http.StatusForbidden, "missing CSRF cookie")
				return
			}

			if !ValidateCSRFToken(cookie.Value, headerToken) {
				writeJSONError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
