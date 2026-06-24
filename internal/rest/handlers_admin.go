package rest

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
	userDir := filepath.Join(s.cfg.Root, "users", req.Username)
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
	if err := s.orch.CascadeDeleteUser(r.Context(), userID); err != nil {
		s.logger.Error("admin delete user: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logger.Info("admin: user deleted", logging.F("username", user.Username))
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (s *Server) handleAdminListRegistries(w http.ResponseWriter, r *http.Request) {
	registries, err := s.db.ListRegistries()
	if err != nil {
		s.logger.Error("admin list registries: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var result []map[string]interface{}
	for _, reg := range registries {
		result = append(result, map[string]interface{}{
			"id":         reg.ID,
			"prefix":     reg.Prefix,
			"created_at": reg.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

type createRegistryRequest struct {
	Prefix string `json:"prefix"`
}

func (s *Server) handleAdminCreateRegistry(w http.ResponseWriter, r *http.Request) {
	var req createRegistryRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prefix == "" {
		writeError(w, http.StatusBadRequest, "prefix is required")
		return
	}

	id := uuid.NewString()
	if err := s.db.InsertRegistry(id, req.Prefix); err != nil {
		s.logger.Error("admin create registry: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.logger.Info("admin: registry added", logging.F("prefix", req.Prefix))
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":     id,
		"prefix": req.Prefix,
	})
}

func (s *Server) handleAdminDeleteRegistry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing registry id")
		return
	}

	if err := s.db.DeleteRegistry(id); err != nil {
		s.logger.Error("admin delete registry: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "registry deleted"})
}

func (s *Server) handleAdminListLogs(w http.ResponseWriter, r *http.Request) {
	logDir := filepath.Join(s.cfg.Root, "core/logs")
	files, err := os.ReadDir(logDir)
	if err != nil {
		s.logger.Error("admin list logs: failed to read logs dir", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to read logs directory")
		return
	}

	var logFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "magicbox-") && strings.HasSuffix(f.Name(), ".log") {
			logFiles = append(logFiles, f.Name())
		}
	}

	if len(logFiles) == 0 {
		writeJSON(w, http.StatusOK, []string{})
		return
	}

	// Alphabetical sorting sorts YYYY-MM-DD format chronologically.
	sort.Strings(logFiles)
	latestLogFile := filepath.Join(logDir, logFiles[len(logFiles)-1])

	file, err := os.Open(latestLogFile)
	if err != nil {
		s.logger.Error("admin list logs: failed to open log file", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to open log file")
		return
	}
	defer file.Close()

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	levelFilter := strings.ToUpper(r.URL.Query().Get("level"))

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if levelFilter != "" {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err == nil {
				if lvl, ok := entry["level"].(string); ok && lvl == levelFilter {
					lines = append(lines, line)
				}
			}
		} else {
			lines = append(lines, line)
		}
	}

	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	writeJSON(w, http.StatusOK, lines)
}
