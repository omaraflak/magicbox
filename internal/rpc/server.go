// Package rpc provides the gRPC server for Magicbox inter-app communication.
package rpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/magicbox/core/internal/p2p"
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
	p2pService   p2p.Service
	grpcServer   *grpc.Server
}

// NewRPCServer creates a new RPCServer with the given dependencies.
func NewRPCServer(database *db.DB, dockerClient *docker.Client, orchestrator *core.Orchestrator, logger *logging.Logger, config *config.Config, p2pService p2p.Service) *RPCServer {
	return &RPCServer{
		db:           database,
		docker:       dockerClient,
		orchestrator: orchestrator,
		logger:       logger,
		cfg:          config,
		jwtSecret:    config.JWTSecret,
		rateLimiter:  NewRateLimiter(),
		p2pService:   p2pService,
	}
}

// Start begins listening on the given port for gRPC connections.
func (s *RPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("grpc: failed to listen on port %s: %w", port, err)
	}

	const maxMessageSize = 512 * 1024 * 1024 // 512 MB
	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.authInterceptor),
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
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

	// 1. Parse unverified token to extract app_id and user_id claims.
	unverifiedClaims, err := rest.ParseAppTokenUnverified(tokenStr)
	if err != nil {
		s.logger.Error("gRPC Auth Fail (Unverified Parse)", logging.F("error", err.Error()))
		return nil, status.Errorf(codes.Unauthenticated, "invalid token format: %v", err)
	}

	// 2. Fetch the app token secret from the database.
	appToken, err := s.db.GetAppToken(unverifiedClaims.AppID, unverifiedClaims.UserID)
	if err != nil {
		s.logger.Error("gRPC Auth Fail (DB Query)", logging.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to query token secret: %v", err)
	}
	if appToken == nil {
		s.logger.Error("gRPC Auth Fail (Token Missing)", logging.F("app", unverifiedClaims.AppID), logging.F("user", unverifiedClaims.UserID))
		return nil, status.Error(codes.Unauthenticated, "app token not found in database")
	}

	// 3. Verify the token signature with the app-specific TokenSecret.
	claims, err := rest.ValidateAppToken([]byte(appToken.TokenSecret), tokenStr)
	if err != nil {
		s.logger.Error("gRPC Auth Fail (Signature)", logging.F("error", err.Error()))
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
		"local",
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
// SendToContact
// ---------------------------------------------------------------------------

func (s *RPCServer) SendToContact(ctx context.Context, req *pb.SendToContactRequest) (*pb.SendToContactResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	// Fetch contact from DB
	contact, err := s.db.GetContactByID(req.ContactId, claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database query error: %v", err)
	}
	if contact == nil {
		return nil, status.Errorf(codes.NotFound, "contact %q not found", req.ContactId)
	}

	targetUserID := contact.TargetUserID

	// Check if the contact's peer ID matches our local host ID (loopback/local transfer).
	// The multiaddr is stored as a base64-encoded invite link, so we decode the payload
	// to extract the raw multiaddr and check if it contains our local host peer ID.
	isLocal := false
	if strings.HasPrefix(contact.Multiaddr, "magicbox://invite/") {
		b64 := strings.TrimPrefix(contact.Multiaddr, "magicbox://invite/")
		if payloadBytes, err := base64.URLEncoding.DecodeString(b64); err == nil {
			var payload struct {
				Multiaddr string `json:"multiaddr"`
			}
			if err := json.Unmarshal(payloadBytes, &payload); err == nil {
				isLocal = strings.Contains(payload.Multiaddr, s.p2pService.HostID())
			}
		}
	}
	if isLocal {
		s.logger.Info("SendToContact: target is a local contact, routing locally",
			logging.F("user_id", claims.UserID),
			logging.F("target_user_id", targetUserID),
		)
		_, err = s.orchestrator.DispatchWebhook(
			ctx,
			req.AppId,
			targetUserID,
			claims.AppID,
			claims.UserID,
			"local",
			req.Payload,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "local webhook dispatch failed: %v", err)
		}
		return &pb.SendToContactResponse{
			Success:       true,
			StatusMessage: "payload delivered locally",
		}, nil
	}

	// Dispatch message directly to the remote peer multiaddress over libp2p
	msg := &p2p.Message{
		AppID:        req.AppId,
		TargetUserID: targetUserID,
		Payload:      req.Payload,
	}

	err = s.p2pService.SendTo(ctx, contact.Multiaddr, msg)
	if err != nil {
		s.logger.Error("SendToContact failed",
			logging.F("user_id", claims.UserID),
			logging.F("contact_id", req.ContactId),
			logging.F("error", err.Error()))
		return &pb.SendToContactResponse{
			Success:       false,
			StatusMessage: fmt.Sprintf("failed to send: %v", err),
		}, nil
	}

	return &pb.SendToContactResponse{
		Success:       true,
		StatusMessage: "message dispatched successfully",
	}, nil
}

// ---------------------------------------------------------------------------
// ListContacts
// ---------------------------------------------------------------------------

func (s *RPCServer) ListContacts(ctx context.Context, _ *pb.ListContactsRequest) (*pb.ListContactsResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if !hasScope(claims.Scopes, "contacts:read") {
		return nil, status.Error(codes.PermissionDenied, "missing scope: contacts:read")
	}

	contacts, err := s.db.GetContacts(claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database query error: %v", err)
	}

	var pbContacts []*pb.Contact
	for _, c := range contacts {
		pbContacts = append(pbContacts, &pb.Contact{
			Id:           c.ID,
			DisplayName:  c.DisplayName,
			Multiaddr:    c.Multiaddr,
			TargetUserId: c.TargetUserID,
		})
	}

	return &pb.ListContactsResponse{Contacts: pbContacts}, nil
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
