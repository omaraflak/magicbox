package rest

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
)

// Server holds all dependencies needed by the REST API handlers.
type Server struct {
	cfg    *config.Config
	db     *db.DB
	docker *docker.Client
	logger *logging.Logger
	orch   *core.Orchestrator
}

// NewServer creates a new REST API server with the given dependencies.
func NewServer(cfg *config.Config, database *db.DB, dockerClient *docker.Client, logger *logging.Logger, orch *core.Orchestrator) *Server {
	return &Server{
		cfg:    cfg,
		db:     database,
		docker: dockerClient,
		logger: logger,
		orch:   orch,
	}
}

// Handler returns the root http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public routes — no authentication required.
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/setup", s.handleSetup)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/csrf", s.handleCSRFToken)

	// Authenticated routes.
	auth := AuthMiddleware(s.cfg.JWTSecret)

	mux.Handle("GET /api/v1/me", auth(http.HandlerFunc(s.handleMe)))
	mux.Handle("POST /api/v1/me/password", auth(http.HandlerFunc(s.handleUpdatePassword)))
	mux.Handle("GET /api/v1/apps", auth(http.HandlerFunc(s.handleListApps)))
	mux.Handle("POST /api/v1/apps/install", auth(http.HandlerFunc(s.handleInstallApp)))
	mux.Handle("DELETE /api/v1/apps/{id}", auth(http.HandlerFunc(s.handleUninstallApp)))
	mux.Handle("POST /api/v1/apps/{id}/start", auth(http.HandlerFunc(s.handleStartApp)))
	mux.Handle("POST /api/v1/apps/{id}/stop", auth(http.HandlerFunc(s.handleStopApp)))
	mux.Handle("POST /api/v1/apps/{id}/update", auth(http.HandlerFunc(s.handleUpdateApp)))
	mux.Handle("POST /api/v1/apps/{id}/rebuild", auth(http.HandlerFunc(s.handleRebuildApp)))
	mux.Handle("POST /api/v1/apps/{id}/rotate-token", auth(http.HandlerFunc(s.handleRotateToken)))

	// Admin-only routes — require both AuthMiddleware and AdminMiddleware.
	admin := AdminMiddleware()

	mux.Handle("GET /api/v1/admin/users", auth(admin(http.HandlerFunc(s.handleAdminListUsers))))
	mux.Handle("POST /api/v1/admin/users", auth(admin(http.HandlerFunc(s.handleAdminCreateUser))))
	mux.Handle("DELETE /api/v1/admin/users/{id}", auth(admin(http.HandlerFunc(s.handleAdminDeleteUser))))
	mux.Handle("GET /api/v1/admin/registries", auth(admin(http.HandlerFunc(s.handleAdminListRegistries))))
	mux.Handle("POST /api/v1/admin/registries", auth(admin(http.HandlerFunc(s.handleAdminCreateRegistry))))
	mux.Handle("DELETE /api/v1/admin/registries/{id}", auth(admin(http.HandlerFunc(s.handleAdminDeleteRegistry))))
	mux.Handle("GET /api/v1/admin/logs", auth(admin(http.HandlerFunc(s.handleAdminListLogs))))

	// Route app traffic through Go server to Traefik if accessed on the dashboard port (helps with single-port proxies / Uberproxy)
	traefikHost := "127.0.0.1:80"
	if _, err := os.Stat("/.dockerenv"); err == nil {
		traefikHost = "magicbox_traefik:80"
	}
	traefikURL, _ := url.Parse("http://" + traefikHost)
	proxy := httputil.NewSingleHostReverseProxy(traefikURL)
	mux.Handle("/u/", proxy)

	// Static file fallback — serve web UI with SPA fallback.
	// For client-side routing, serve index.html for any path that doesn't
	// match a real static file (e.g. /admin/logs → index.html).
	webDir := s.cfg.Root + "/core/web/"
	fileServer := http.FileServer(http.Dir(webDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists on disk.
		filePath := webDir + r.URL.Path
		if r.URL.Path != "/" {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				// Not a real file — serve index.html for SPA routing.
				http.ServeFile(w, r, webDir+"index.html")
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	}))

	// Wrap everything with security headers.
	return SecurityHeadersMiddleware(mux)
}
