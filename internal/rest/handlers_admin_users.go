package rest

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/magicbox/core/internal/logging"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		s.logger.Error("admin list users: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var result []map[string]interface{}
	for _, u := range users {
		result = append(result, map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"is_admin":   u.IsAdmin,
			"created_at": u.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
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

	existing, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := uuid.NewString()
	if err := s.db.CreateUser(userID, req.Username, string(hash), req.IsAdmin); err != nil {
		s.logger.Error("admin create user: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create user directories.
	userDir := filepath.Join(s.config.Root, "users", req.Username)
	for _, sub := range []string{"apps", "shared"} {
		if err := os.MkdirAll(filepath.Join(userDir, sub), 0750); err != nil {
			s.logger.Error("admin create user: failed to create directory", logging.F("error", err.Error()))
		}
	}

	s.logger.Info("admin: user created", logging.F("username", req.Username))
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       userID,
		"username": req.Username,
		"is_admin": req.IsAdmin,
	})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Cascade delete via orchestrator.
	if err := s.orchestrator.CascadeDeleteUser(r.Context(), userID); err != nil {
		s.logger.Error("admin delete user: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logger.Info("admin: user deleted", logging.F("username", user.Username))
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
