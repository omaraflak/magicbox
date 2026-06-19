// Package rpc provides the gRPC server for Magicbox inter-app communication.
package rpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
)

// RPCServer holds the gRPC server and its dependencies.
type RPCServer struct {
	db           *db.DB
	docker       *docker.Client
	orchestrator *core.Orchestrator
	logger       *logging.Logger
	jwtSecret    []byte
	grpcServer   *grpc.Server
}

// NewRPCServer creates a new RPCServer with the given dependencies.
func NewRPCServer(database *db.DB, dockerClient *docker.Client, orch *core.Orchestrator, logger *logging.Logger, cfg *config.Config) *RPCServer {
	return &RPCServer{
		db:           database,
		docker:       dockerClient,
		orchestrator: orch,
		logger:       logger,
		jwtSecret:    cfg.JWTSecret,
	}
}

// Start begins listening on the given port for gRPC connections.
// TODO: Implement the MagicboxOS service once proto compilation is available.
// The proto service definition should include methods for:
//   - SendWebhook: allows apps to dispatch webhooks to other apps
//   - ReadFile / WriteFile: scoped file access for apps
//   - GetProfile: read user profile information
//   - ListSharedVolumes: discover available shared volumes
func (s *RPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("grpc: failed to listen on port %s: %w", port, err)
	}

	s.grpcServer = grpc.NewServer()

	// TODO: Register the MagicboxOS service implementation here.
	// Example: pb.RegisterMagicboxOSServer(s.grpcServer, s)

	s.logger.Info("gRPC server listening", logging.F("port", port))
	return s.grpcServer.Serve(lis)
}

// GracefulStop stops the gRPC server gracefully.
func (s *RPCServer) GracefulStop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}
