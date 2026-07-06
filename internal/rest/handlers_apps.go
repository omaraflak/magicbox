package rest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
)

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

	statusFilter := r.URL.Query().Get("status")
	var result []map[string]interface{}
	for _, a := range apps {
		if statusFilter == "" || a.Status == statusFilter {
			result = append(result, appResponse(a))
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func appResponse(app db.App) map[string]interface{} {
	return map[string]interface{}{
		"id":         app.ID,
		"app_id":     app.AppID,
		"name":       app.Name,
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

	app, err := s.orchestrator.Install(r.Context(), claims.UserID, manifestData)
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
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	wipe := (r.URL.Query().Get("wipe") == "true")

	if err := s.orchestrator.Uninstall(r.Context(), app.ID, wipe); err != nil {
		s.logger.Error("uninstall app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app uninstalled"})
}

func (s *Server) handleStartApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	if err := s.orchestrator.Start(r.Context(), app.ID); err != nil {
		s.logger.Error("start app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app started"})
}

func (s *Server) handleStopApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	if err := s.orchestrator.Stop(r.Context(), app.ID); err != nil {
		s.logger.Error("stop app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app stopped"})
}

func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxBodySize)
	defer body.Close()
	manifestData, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	if err := s.orchestrator.Update(r.Context(), app.ID, manifestData); err != nil {
		s.logger.Error("update app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app updated"})
}

func (s *Server) handleRebuildApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	if err := s.orchestrator.Rebuild(r.Context(), app.ID); err != nil {
		s.logger.Error("rebuild app: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "app rebuilt"})
}

func (s *Server) handleRotateToken(w http.ResponseWriter, r *http.Request) {
	app, ok := s.getOwnedApp(w, r)
	if !ok {
		return
	}

	if err := s.orchestrator.RotateToken(r.Context(), app.ID); err != nil {
		s.logger.Error("rotate token: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "token rotated"})
}

// handleAppProxy dynamically reverse proxies requests to the target app's container IP.
func (s *Server) handleAppProxy(w http.ResponseWriter, r *http.Request) {
	// Path format: /u/{username}/{routeSlug}/...
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[1] != "u" {
		writeError(w, http.StatusBadRequest, "invalid app path")
		return
	}

	username := parts[2]
	routeSlug := parts[3]

	// Redirect to trailing slash if missing (e.g. /u/omar/drive -> /u/omar/drive/) to prevent relative asset path resolution issues.
	if len(parts) == 4 {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	s.logger.Info("handleAppProxy routing request", logging.F("path", r.URL.Path), logging.F("parts", fmt.Sprintf("%+v", parts)))

	// Fetch target user and app record
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query user: "+err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	app, err := s.db.GetAppByRouteSlugAndUserID(routeSlug, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query app: "+err.Error())
		return
	}
	if app == nil {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}
	if app.Status != "running" || app.ContainerID == "" {
		writeError(w, http.StatusServiceUnavailable, "app is not running")
		return
	}

	if s.docker == nil {
		writeError(w, http.StatusInternalServerError, "docker client not initialized (mock)")
		return
	}

	// Inspect container to resolve IP address
	status, err := s.docker.InspectContainer(r.Context(), app.ContainerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve container IP: "+err.Error())
		return
	}
	if status.IPAddress == "" {
		writeError(w, http.StatusServiceUnavailable, "app container has no IP address")
		return
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", status.IPAddress, app.EntryPort))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse app target URL: "+err.Error())
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Strip prefix /u/{username}/{routeSlug} before sending to the app container.
	prefix := fmt.Sprintf("/u/%s/%s", username, routeSlug)
	r.Header.Set("X-Forwarded-Prefix", prefix)
	r.Header.Set("X-Original-URI", r.URL.Path)
	r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(w, r)
}

// getOwnedApp extracts claims, validates the path ID, retrieves the app,
// and validates that the logged-in user owns the app.
func (s *Server) getOwnedApp(w http.ResponseWriter, r *http.Request) (*db.App, bool) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil, false
	}

	appDBID := r.PathValue("id")
	if appDBID == "" {
		writeError(w, http.StatusBadRequest, "missing app id")
		return nil, false
	}

	app, err := s.db.GetAppByID(appDBID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return nil, false
	}
	if app == nil || app.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "app not found")
		return nil, false
	}

	return app, true
}

func (s *Server) handleListPendingPermissions(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requests := s.orchestrator.ListPendingPermissions(claims.UserID)
	writeJSON(w, http.StatusOK, requests)
}

func (s *Server) handleApprovePermissionRequest(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	reqID := r.PathValue("id")
	if reqID == "" {
		writeError(w, http.StatusBadRequest, "missing request id")
		return
	}

	ok := s.orchestrator.ApprovePermissionRequest(reqID)
	if !ok {
		writeError(w, http.StatusNotFound, "pending request not found or decision already sent")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "permission request approved"})
}

func (s *Server) handleRejectPermissionRequest(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	reqID := r.PathValue("id")
	if reqID == "" {
		writeError(w, http.StatusBadRequest, "missing request id")
		return
	}

	ok := s.orchestrator.RejectPermissionRequest(reqID)
	if !ok {
		writeError(w, http.StatusNotFound, "pending request not found or decision already sent")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "permission request rejected"})
}

