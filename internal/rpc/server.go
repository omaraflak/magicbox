// Package rpc provides the gRPC server for Magicbox inter-app communication.
package rpc

import (
	"context"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/rest"
)

// appClaimsKey is the context key for validated AppTokenClaims.
type appClaimsKey struct{}

// RPCServer implements the MagicboxOS gRPC service.
type RPCServer struct {
	pb.UnimplementedMagicboxOSServer

	db           *db.DB
	docker       *docker.Client
	orchestrator *core.Orchestrator
	logger       *logging.Logger
	cfg          *config.Config
	jwtSecret    []byte
	rateLimiter  *RateLimiter
	grpcServer   *grpc.Server
}

// NewRPCServer creates a new RPCServer with the given dependencies.
func NewRPCServer(database *db.DB, dockerClient *docker.Client, orch *core.Orchestrator, logger *logging.Logger, cfg *config.Config) *RPCServer {
	return &RPCServer{
		db:           database,
		docker:       dockerClient,
		orchestrator: orch,
		logger:       logger,
		cfg:          cfg,
		jwtSecret:    cfg.JWTSecret,
		rateLimiter:  NewRateLimiter(),
	}
}

// Start begins listening on the given port for gRPC connections.
func (s *RPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("grpc: failed to listen on port %s: %w", port, err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.authInterceptor),
	)

	pb.RegisterMagicboxOSServer(s.grpcServer, s)

	s.logger.Info("gRPC server listening", logging.F("port", port))
	return s.grpcServer.Serve(lis)
}

// GracefulStop stops the gRPC server gracefully.
func (s *RPCServer) GracefulStop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// ---------------------------------------------------------------------------
// Auth interceptor: validates JWT from metadata and enforces rate limits.
// ---------------------------------------------------------------------------

func (s *RPCServer) authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Extract token from gRPC metadata.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authValues := md.Get("authorization")
	if len(authValues) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	tokenStr := authValues[0]
	// Strip "Bearer " prefix if present.
	tokenStr = strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer "))

	claims, err := rest.ValidateAppToken(s.jwtSecret, tokenStr)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Rate limit by app_id.
	if !s.rateLimiter.Allow(claims.AppID) {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	// Inject claims into context.
	ctx = context.WithValue(ctx, appClaimsKey{}, claims)
	return handler(ctx, req)
}

// claimsFromContext extracts the validated AppTokenClaims from the context.
func claimsFromContext(ctx context.Context) *rest.AppTokenClaims {
	claims, _ := ctx.Value(appClaimsKey{}).(*rest.AppTokenClaims)
	return claims
}

// ---------------------------------------------------------------------------
// SendWebhook
// ---------------------------------------------------------------------------

func (s *RPCServer) SendWebhook(ctx context.Context, req *pb.SendWebhookRequest) (*pb.SendWebhookResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	httpStatus, err := s.orchestrator.DispatchWebhook(
		ctx,
		req.TargetAppId,
		req.TargetUserId,
		claims.AppID,
		claims.UserID,
		req.Payload,
	)
	if err != nil {
		s.logger.Error("grpc SendWebhook failed",
			logging.F("source_app", claims.AppID),
			logging.F("target_app", req.TargetAppId),
			logging.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "webhook dispatch failed: %v", err)
	}

	return &pb.SendWebhookResponse{HttpStatus: int32(httpStatus)}, nil
}

// ---------------------------------------------------------------------------
// GetProfile
// ---------------------------------------------------------------------------

func (s *RPCServer) GetProfile(ctx context.Context, _ *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if !hasScope(claims.Scopes, "profile:read") {
		return nil, status.Error(codes.PermissionDenied, "missing scope: profile:read")
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &pb.GetProfileResponse{
		UserId:    user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// ListSharedVolumes
// ---------------------------------------------------------------------------

func (s *RPCServer) ListSharedVolumes(ctx context.Context, _ *pb.ListSharedVolumesRequest) (*pb.ListSharedVolumesResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	var volumes []*pb.SharedVolume
	for _, scope := range claims.Scopes {
		volName, readOnly, ok := core.ScopeToVolumeAccess(scope)
		if ok {
			access := "rw"
			if readOnly {
				access = "ro"
			}
			volumes = append(volumes, &pb.SharedVolume{
				Name:   volName,
				Access: access,
			})
		}
	}

	return &pb.ListSharedVolumesResponse{Volumes: volumes}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// hasScope checks if a scope is present in a list.
func hasScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}
