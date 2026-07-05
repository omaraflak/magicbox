package rest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/magicbox/core/internal/config"
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

	userID, err := s.createAdminUser(w, req.Username, req.Password)
	if err != nil {
		s.logger.Error("setup failed", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Info("setup: admin user created", logging.F("username", req.Username))
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       userID,
		"username": req.Username,
	})
}

type setupRecoverRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Mnemonic string `json:"mnemonic"`
}

func (s *Server) handleSetupRecover(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.UserCount()
	if err != nil {
		s.logger.Error("setup recover: failed to count users", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if count > 0 {
		writeError(w, http.StatusForbidden, "setup already completed")
		return
	}

	var req setupRecoverRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := config.RecoverKeys(s.config.Root, req.Mnemonic, 0, 0); err != nil {
		s.logger.Error("setup recover: failed to recover keys", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, "failed to recover keys: "+err.Error())
		return
	}

	userID, err := s.createAdminUser(w, req.Username, req.Password)
	if err != nil {
		s.logger.Error("setup recover failed", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Info("setup recover: admin user created and keys recovered", logging.F("username", req.Username))
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

	token, err := GenerateSessionToken(s.config.JWTSecret, user.ID, user.Username, user.IsAdmin)
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

// createAdminUser validates the username/password, hashes the password, inserts
// the user into the database, creates system directories on disk, and sets up
// auto-login cookies.
func (s *Server) createAdminUser(w http.ResponseWriter, username, password string) (string, error) {
	if !usernameRegex.MatchString(username) {
		return "", fmt.Errorf("username must be 3-32 lowercase alphanumeric or underscore characters")
	}
	if len(password) < minPassLen {
		return "", fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.NewString()
	if err := s.db.CreateUser(userID, username, string(hash), true); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// Create user directories.
	userDir := filepath.Join(s.config.Root, "users", username)
	for _, sub := range []string{"apps", "shared"} {
		if err := os.MkdirAll(filepath.Join(userDir, sub), 0750); err != nil {
			s.logger.Error("setup: failed to create user directory", logging.F("error", err.Error()))
		}
	}

	// Auto-login: generate session token and set cookie.
	token, err := GenerateSessionToken(s.config.JWTSecret, userID, username, true)
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	SetSessionCookie(w, token)

	return userID, nil
}
