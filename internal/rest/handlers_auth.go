package rest

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/magicbox/core/internal/logging"
)

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.UserCount()
	if err != nil {
		s.logger.Error("setup: failed to count users", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if count > 0 {
		writeError(w, http.StatusForbidden, "setup already completed")
		return
	}

	var req setupRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !usernameRegex.MatchString(req.Username) {
		writeError(w, http.StatusBadRequest, "username must be 3-32 lowercase alphanumeric or underscore characters")
		return
	}
	if len(req.Password) < minPassLen {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		s.logger.Error("setup: failed to hash password", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := uuid.NewString()
	if err := s.db.CreateUser(userID, req.Username, string(hash), true); err != nil {
		s.logger.Error("setup: failed to create user", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create user directories.
	userDir := filepath.Join(s.cfg.Root, "users", req.Username)
	for _, sub := range []string{"apps", "shared"} {
		if err := os.MkdirAll(filepath.Join(userDir, sub), 0750); err != nil {
			s.logger.Error("setup: failed to create user directory", logging.F("error", err.Error()))
		}
	}

	// Auto-login: generate session token and set cookie.
	token, err := GenerateSessionToken(s.cfg.JWTSecret, userID, req.Username, true)
	if err != nil {
		s.logger.Error("setup: failed to generate session token", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	SetSessionCookie(w, token)

	s.logger.Info("setup: admin user created", logging.F("username", req.Username))
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       userID,
		"username": req.Username,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		s.logger.Error("login: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := GenerateSessionToken(s.cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		s.logger.Error("login: failed to generate token", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	SetSessionCookie(w, token)

	s.logger.Info("login: user authenticated", logging.F("username", req.Username))
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged in"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := GenerateCSRFToken()
	if err != nil {
		s.logger.Error("csrf: failed to generate token", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JS needs to read this.
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"is_admin":   user.IsAdmin,
		"created_at": user.CreatedAt,
	})
}

func (s *Server) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid current password")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := s.db.UpdateUserPassword(claims.UserID, string(hash)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}
