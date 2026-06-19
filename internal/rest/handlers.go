package rest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
)

const (
	maxBodySize  = 1 << 20 // 1 MB
	bcryptCost   = 12
	minPassLen   = 8
	minUserLen   = 3
	maxUserLen   = 32
)

var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func readJSON(r *http.Request, v interface{}) error {
	body := http.MaxBytesReader(nil, r.Body, maxBodySize)
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

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
	token, err := GenerateSessionToken(s.cfg.JWTSecret, userID, true)
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

// ---------------------------------------------------------------------------
// Auth: Login / Logout / CSRF
// ---------------------------------------------------------------------------

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

	token, err := GenerateSessionToken(s.cfg.JWTSecret, user.ID, user.IsAdmin)
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

// ---------------------------------------------------------------------------
// User: Me
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Apps: List, Install, Uninstall, Start, Stop, Update, RotateToken
// ---------------------------------------------------------------------------

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	apps, err := s.db.ListAppsByUserID(claims.UserID)
	if err != nil {
		s.logger.Error("list apps: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Optional status filter.
	statusFilter := r.URL.Query().Get("status")
	if statusFilter != "" {
		var filtered []interface{}
		for _, a := range apps {
			if a.Status == statusFilter {
				filtered = append(filtered, appResponse(a))
			}
		}
		writeJSON(w, http.StatusOK, filtered)
		return
	}

	var result []interface{}
	for _, a := range apps {
		result = append(result, appResponse(a))
	}
	writeJSON(w, http.StatusOK, result)
}

func appResponse(a interface{}) map[string]interface{} {
	var app db.App
	switch val := a.(type) {
	case db.App:
		app = val
	case *db.App:
		if val != nil {
			app = *val
		}
	default:
		return nil
	}

	return map[string]interface{}{
		"id":         app.ID,
		"app_id":     app.AppID,
		"status":     app.Status,
		"image":      app.Image,
		"version":    app.Version,
		"route_slug": app.RouteSlug,
		"host":       app.Host,
	}
}

func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxBodySize)
	defer body.Close()
	manifestData, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	app, err := s.orch.Install(r.Context(), claims.UserID, manifestData)
	if err != nil {
		s.logger.Error("install app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        app.ID,
		"app_id":    app.AppID,
		"status":    app.Status,
		"image":     app.Image,
		"version":   app.Version,
		"route_slug": app.RouteSlug,
	})
}

func (s *Server) handleUninstallApp(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	appDBID := r.PathValue("id")
	if appDBID == "" {
		writeError(w, http.StatusBadRequest, "missing app id")
		return
	}

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	if err := s.orch.Uninstall(r.Context(), appDBID); err != nil {
		s.logger.Error("uninstall app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app uninstalled"})
}

func (s *Server) handleStartApp(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	appDBID := r.PathValue("id")

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	if err := s.orch.Start(r.Context(), appDBID); err != nil {
		s.logger.Error("start app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app started"})
}

func (s *Server) handleStopApp(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	appDBID := r.PathValue("id")

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	if err := s.orch.Stop(r.Context(), appDBID); err != nil {
		s.logger.Error("stop app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app stopped"})
}

func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	appDBID := r.PathValue("id")

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxBodySize)
	defer body.Close()
	manifestData, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	if err := s.orch.Update(r.Context(), appDBID, manifestData); err != nil {
		s.logger.Error("update app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app updated"})
}

func (s *Server) handleRotateToken(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	appDBID := r.PathValue("id")

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	if err := s.orch.RotateToken(r.Context(), appDBID); err != nil {
		s.logger.Error("rotate token: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "token rotated"})
}

// ---------------------------------------------------------------------------
// Admin: Users
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Admin: Registries
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Admin: Logs
// ---------------------------------------------------------------------------

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
