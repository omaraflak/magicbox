// Package main is the entry point for the Magicbox server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/cron"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/rest"
	"github.com/magicbox/core/internal/rpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

func run() error {
	// 1. Load configuration.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Initialize logger.
	logger, err := logging.New(filepath.Join(cfg.Root, "core", "logs"))
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()
	logger.Info("starting Magicbox server", logging.F("port", cfg.Port))

	// 3. Open database and run migrations.
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	logger.Info("database initialized")

	// 4. Initialize Docker client.
	dockerClient, err := docker.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker client: %w", err)
	}
	defer dockerClient.Close()
	logger.Info("Docker client connected")

	// 5. Ensure magicbox_net network exists.
	ctx := context.Background()
	if err := dockerClient.EnsureNetwork(ctx); err != nil {
		return fmt.Errorf("failed to ensure Docker network: %w", err)
	}
	logger.Info("Docker network ready")

	// 6. Ensure Traefik container is running.
	if err := dockerClient.EnsureTraefik(ctx); err != nil {
		return fmt.Errorf("failed to ensure Traefik: %w", err)
	}
	logger.Info("Traefik proxy ready")

	// 7. Create orchestrator.
	orch := core.NewOrchestrator(database, dockerClient, cfg, logger, rest.GenerateAppToken)

	// 8. Start cron jobs.
	stopTransit := cron.StartTransitCleaner(cfg.Root, logger)
	stopBackup := cron.StartBackupJob(cfg.DBPath, filepath.Join(cfg.Root, "backups"), logger)
	logger.Info("cron jobs started")

	// 9. Initialize and start P2P service.
	p2pKey, err := p2p.ParsePEMToPrivKey(cfg.PrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse identity key for P2P: %w", err)
	}

	p2pService := p2p.NewLibp2pService(p2pKey, nil, logger)
	if err := p2pService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start P2P service: %w", err)
	}
	defer p2pService.Stop()

	// Register generic P2P routing callback handler that forwards received P2P payloads to target app webhooks.
	p2pService.SetDefaultHandler(func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		if msg.TargetUserID == "" {
			logger.Warn("Incoming P2P message dropped: missing target_user_id in message envelope",
				logging.F("from_peer", fromPeerID),
				logging.F("protocol", msg.ProtocolType),
			)
			return fmt.Errorf("missing target_user_id")
		}

		logger.Info("Routing incoming P2P message to app webhook",
			logging.F("from_peer", fromPeerID),
			logging.F("target_user", msg.TargetUserID),
			logging.F("app_id", msg.ProtocolType),
		)

		// Dispatch message payload to the local app's container webhook endpoint.
		// Set source app ID as "p2p-gateway" and source user ID as "peer:" + fromPeerID.
		_, err := orch.DispatchWebhook(
			ctx,
			msg.ProtocolType,
			msg.TargetUserID,
			"p2p-gateway",
			"peer:"+fromPeerID,
			msg.Payload,
		)
		if err != nil {
			logger.Error("Failed to dispatch incoming P2P message webhook",
				logging.F("from_peer", fromPeerID),
				logging.F("target_user", msg.TargetUserID),
				logging.F("app_id", msg.ProtocolType),
				logging.F("error", err.Error()),
			)
			return err
		}
		return nil
	})

	// 10. Start gRPC server in a goroutine.
	rpcServer := rpc.NewRPCServer(database, dockerClient, orch, logger, cfg, p2pService)
	go func() {
		if err := rpcServer.Start("50051"); err != nil {
			logger.Error("gRPC server error", logging.F("error", err.Error()))
		}
	}()

	// 11. Create REST server and build handler.
	restServer := rest.NewServer(cfg, database, dockerClient, logger, orch, p2pService)
	handler := restServer.Handler()

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 11. Start HTTP server.
	go func() {
		logger.Info("HTTP server listening", logging.F("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", logging.F("error", err.Error()))
		}
	}()

	// 12. Handle graceful shutdown on SIGTERM/SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	logger.Info("shutdown signal received", logging.F("signal", sig.String()))

	// Stop HTTP server with 15s timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", logging.F("error", err.Error()))
	}

	// Stop gRPC server.
	rpcServer.GracefulStop()

	// Stop cron jobs.
	stopTransit()
	stopBackup()

	logger.Info("shutdown complete")
	return nil
}
