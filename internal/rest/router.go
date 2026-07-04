package rest

import (
	"net/http"
	"os"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// Server holds all dependencies needed by the REST API handlers.
type Server struct {
	config       *config.Config
	db           *db.DB
	docker       *docker.Client
	logger       *logging.Logger
	orchestrator *core.Orchestrator
	p2pService   p2p.Service
	onRestart    func()
}

// NewServer creates a new REST API server with the given dependencies.
func NewServer(config *config.Config, db *db.DB, dockerClient *docker.Client, logger *logging.Logger, orchestrator *core.Orchestrator, p2pService p2p.Service) *Server {
	return &Server{
		config:       config,
		db:           db,
		docker:       dockerClient,
		logger:       logger,
		orchestrator: orchestrator,
		p2pService:   p2pService,
		onRestart:    func() { os.Exit(0) },
	}
}

// Handler returns the root http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public routes — no authentication required.
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/setup", s.handleSetup)
	mux.HandleFunc("POST /api/v1/setup/recover", s.handleSetupRecover)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/csrf", s.handleCSRFToken)

	// Authenticated routes.
	auth := AuthMiddleware(s.config.JWTSecret)

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

	// Contact routes.
	mux.Handle("GET /api/v1/contacts", auth(http.HandlerFunc(s.handleListContacts)))
	mux.Handle("DELETE /api/v1/contacts/{id}", auth(http.HandlerFunc(s.handleDeleteContact)))
	mux.Handle("POST /api/v1/contacts/request", auth(http.HandlerFunc(s.handleSendContactRequest)))
	mux.Handle("GET /api/v1/contacts/requests", auth(http.HandlerFunc(s.handleListContactRequests)))
	mux.Handle("POST /api/v1/contacts/requests/{id}/accept", auth(http.HandlerFunc(s.handleAcceptContactRequest)))
	mux.Handle("POST /api/v1/contacts/requests/{id}/reject", auth(http.HandlerFunc(s.handleRejectContactRequest)))

	// Contact P2P invitation link
	mux.Handle("GET /api/v1/me/invitation", auth(http.HandlerFunc(s.handleGetInvitation)))

	// Admin-only routes — require both AuthMiddleware and AdminMiddleware.
	admin := AdminMiddleware()

	mux.Handle("GET /api/v1/admin/users", auth(admin(http.HandlerFunc(s.handleAdminListUsers))))
	mux.Handle("POST /api/v1/admin/users", auth(admin(http.HandlerFunc(s.handleAdminCreateUser))))
	mux.Handle("DELETE /api/v1/admin/users/{id}", auth(admin(http.HandlerFunc(s.handleAdminDeleteUser))))
	mux.Handle("GET /api/v1/admin/registries", auth(admin(http.HandlerFunc(s.handleAdminListRegistries))))
	mux.Handle("POST /api/v1/admin/registries", auth(admin(http.HandlerFunc(s.handleAdminCreateRegistry))))
	mux.Handle("DELETE /api/v1/admin/registries/{id}", auth(admin(http.HandlerFunc(s.handleAdminDeleteRegistry))))
	mux.Handle("GET /api/v1/admin/logs", auth(admin(http.HandlerFunc(s.handleAdminListLogs))))
	mux.Handle("POST /api/v1/admin/upgrade", auth(admin(http.HandlerFunc(s.handleAdminUpgrade))))
	mux.Handle("POST /api/v1/admin/restart", auth(admin(http.HandlerFunc(s.handleAdminRestart))))
	mux.Handle("GET /api/v1/admin/mnemonic", auth(admin(http.HandlerFunc(s.handleAdminGetMnemonic))))
	mux.Handle("POST /api/v1/admin/mnemonic/acknowledge", auth(admin(http.HandlerFunc(s.handleAdminAcknowledgeMnemonic))))
	mux.Handle("POST /api/v1/admin/keys/rotate-encryption", auth(admin(http.HandlerFunc(s.handleAdminRotateEncryptionKeys))))
	mux.Handle("POST /api/v1/admin/keys/rotate-identity", auth(admin(http.HandlerFunc(s.handleAdminRotateIdentityKeys))))
	mux.Handle("POST /api/v1/admin/keys/reset-identity", auth(admin(http.HandlerFunc(s.handleAdminResetIdentityKeys))))

	// Dynamically proxy app traffic directly to container IP after authenticating and verifying ownership
	mux.Handle("/u/", auth(AppAccessMiddleware()(http.HandlerFunc(s.handleAppProxy))))

	// Static file fallback — serve web UI with SPA fallback.
	// For client-side routing, serve index.html for any path that doesn't
	// match a real static file (e.g. /admin/logs → index.html).
	webDir := s.config.Root + "/core/web/"
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
