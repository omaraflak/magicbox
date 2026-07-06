// Package rpc provides the gRPC server for Magicbox inter-app communication.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/docker"
	"github.com/magicbox/core/internal/invite"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/protocol"
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
	isLocal := contact.PeerID == s.p2pService.HostID()
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

	err = s.p2pService.SendTo(ctx, contact.Multiaddr, contact.EncPubKey, msg)
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
		// Build invite link from stored contact identity.
		contactInviteLink, _ := invite.Build(&invite.Payload{
			Multiaddr:    c.Multiaddr,
			UserID:       c.TargetUserID,
			EncPubKey:    c.EncPubKey,
			MasterPubKey: c.MasterPubKey,
		})

		pbContacts = append(pbContacts, &pb.Contact{
			Id:           c.ID,
			DisplayName:  c.DisplayName,
			Multiaddr:    c.Multiaddr,
			TargetUserId: c.TargetUserID,
			InviteLink:   contactInviteLink,
			Status:       "active",
		})
	}

	// Also get outgoing contact requests to return as pending contacts
	reqs, err := s.db.GetContactRequests(claims.UserID, "outgoing")
	if err != nil {
		s.logger.Error("ListContacts: failed to get outgoing requests", logging.F("error", err.Error()))
	} else {
		for _, r := range reqs {
			pbContacts = append(pbContacts, &pb.Contact{
				Id:           r.ID,
				DisplayName:  r.DisplayName,
				Multiaddr:    r.Multiaddr,
				TargetUserId: r.TargetUserID,
				InviteLink:   r.Multiaddr, // for pending requests, multiaddr is the invite link itself
				Status:       "pending",
			})
		}
	}

	return &pb.ListContactsResponse{Contacts: pbContacts}, nil
}

// ---------------------------------------------------------------------------
// GetInviteLink
// ---------------------------------------------------------------------------

func (s *RPCServer) GetInviteLink(ctx context.Context, _ *pb.GetInviteLinkRequest) (*pb.GetInviteLinkResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if !hasScope(claims.Scopes, "contacts:read") {
		return nil, status.Error(codes.PermissionDenied, "missing scope: contacts:read")
	}

	peerID := s.p2pService.HostID()
	if peerID == "" {
		return nil, status.Error(codes.Unavailable, "P2P network service unavailable")
	}

	multiaddrs := s.p2pService.Multiaddrs()
	if len(multiaddrs) == 0 {
		return nil, status.Error(codes.Unavailable, "P2P network has no listening addresses")
	}

	targetAddr := p2p.PreferRelayAddr(multiaddrs)

	hexPub, err := s.cfg.Keys.EncPubKeyHex()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse local encryption public key: %v", err)
	}

	hexMasterPub, err := s.cfg.Keys.MasterPubKeyHex()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse local master public key: %v", err)
	}

	payload := &invite.Payload{
		Multiaddr:    targetAddr,
		UserID:       claims.UserID,
		EncPubKey:    hexPub,
		MasterPubKey: hexMasterPub,
	}

	inviteLink, err := invite.Build(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build invitation link: %v", err)
	}

	return &pb.GetInviteLinkResponse{InviteLink: inviteLink}, nil
}

// ---------------------------------------------------------------------------
// SendContactRequest
// ---------------------------------------------------------------------------

func (s *RPCServer) SendContactRequest(ctx context.Context, req *pb.SendContactRequestRequest) (*pb.SendContactRequestResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if !hasScope(claims.Scopes, "contacts:write") {
		return nil, status.Error(codes.PermissionDenied, "missing scope: contacts:write")
	}

	if req.DisplayName == "" || req.InviteLink == "" {
		return nil, status.Error(codes.InvalidArgument, "display_name and invite_link are required")
	}

	// Parse the invite link to extract the remote peer's info.
	payload, err := invite.Parse(req.InviteLink)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid invite link: %v", err)
	}

	peerID := invite.ExtractPeerID(payload.Multiaddr)
	if peerID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid invite link: could not extract peer ID")
	}

	if payload.UserID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid invite link: missing user_id")
	}

	// Check if there is already an incoming request from this user.
	incomingReq, err := s.db.GetIncomingContactRequestByTargetUserID(claims.UserID, payload.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query incoming contact request: %v", err)
	}

	if incomingReq != nil {
		// Incoming request exists! Accept it automatically.
		existing, err := s.db.GetContactByTargetUserID(claims.UserID, incomingReq.TargetUserID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to query existing contact: %v", err)
		}

		var contactID string
		if existing != nil {
			contactID = existing.ID
			if err := s.db.UpdateContactFromRequest(existing.ID, incomingReq.PeerID, incomingReq.Multiaddr, incomingReq.EncPubKey, incomingReq.MasterPubKey); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to update contact: %v", err)
			}
		} else {
			contactID = uuid.NewString()
			if err := s.db.AddContact(
				contactID, claims.UserID, incomingReq.DisplayName,
				incomingReq.PeerID, incomingReq.Multiaddr,
				incomingReq.TargetUserID, incomingReq.EncPubKey, incomingReq.MasterPubKey,
			); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create contact: %v", err)
			}
		}

		// Send accept message back to the requester.
		ourEncPubHex, err := s.cfg.Keys.EncPubKeyHex()
		if err == nil {
			ourMultiaddr := p2p.PreferRelayAddr(s.p2pService.Multiaddrs())
			ourMasterPubHex, err := s.cfg.Keys.MasterPubKeyHex()
			if err == nil {
				user, err := s.db.GetUserByID(claims.UserID)
				if err == nil && user != nil {
					acceptPayload, _ := json.Marshal(protocol.ContactAcceptPayload{
						DisplayName:  user.Username,
						Multiaddr:    ourMultiaddr,
						EncPubKey:    ourEncPubHex,
						UserID:       claims.UserID,
						MasterPubKey: ourMasterPubHex,
					})
					contactForQueue := db.Contact{
						ID:           contactID,
						Multiaddr:    incomingReq.Multiaddr,
						EncPubKey:    incomingReq.EncPubKey,
						TargetUserID: incomingReq.TargetUserID,
					}
					_ = protocol.EnqueueForContacts(s.db, []db.Contact{contactForQueue}, protocol.AppIDContactAccept, acceptPayload)
				}
			}
		}

		// Delete the incoming request.
		_ = s.db.DeleteContactRequest(claims.UserID, incomingReq.ID)

		return &pb.SendContactRequestResponse{
			Success:       true,
			StatusMessage: "Contact request accepted automatically from existing invitation",
		}, nil
	}

	// Store as outgoing request.
	reqID := uuid.NewString()
	if err := s.db.InsertContactRequest(
		reqID, claims.UserID, "outgoing", req.DisplayName,
		peerID, payload.Multiaddr, payload.UserID, payload.EncPubKey, payload.MasterPubKey,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store contact request: %v", err)
	}

	// Build the request payload with our own info.
	ourMultiaddr := p2p.PreferRelayAddr(s.p2pService.Multiaddrs())

	ourEncPubHex, err := s.cfg.Keys.EncPubKeyHex()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read encryption key: %v", err)
	}

	ourMasterPubHex, err := s.cfg.Keys.MasterPubKeyHex()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse local master public key: %v", err)
	}

	// Look up the user's username for the display name in the request payload.
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	requestPayload, _ := json.Marshal(protocol.ContactRequestPayload{
		DisplayName:  user.Username,
		Multiaddr:    ourMultiaddr,
		EncPubKey:    ourEncPubHex,
		UserID:       claims.UserID,
		MasterPubKey: ourMasterPubHex,
	})

	// Enqueue the request message for delivery.
	tempContact := db.Contact{
		ID:           reqID,
		Multiaddr:    payload.Multiaddr,
		EncPubKey:    payload.EncPubKey,
		TargetUserID: payload.UserID,
	}
	if err := protocol.EnqueueForContacts(s.db, []db.Contact{tempContact}, protocol.AppIDContactRequest, requestPayload); err != nil {
		s.logger.Error("failed to enqueue contact request", logging.F("error", err.Error()))
	}

	return &pb.SendContactRequestResponse{
		Success:       true,
		StatusMessage: "Contact request sent",
	}, nil
}

// ---------------------------------------------------------------------------
// RequestPermissions
// ---------------------------------------------------------------------------

func (s *RPCServer) RequestPermissions(ctx context.Context, req *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if len(req.Requests) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one scope request is required")
	}

	var coreRequests []core.ScopeWithReason
	for _, r := range req.Requests {
		if r.Scope == "" {
			return nil, status.Error(codes.InvalidArgument, "scope cannot be empty")
		}
		coreRequests = append(coreRequests, core.ScopeWithReason{
			Scope:  r.Scope,
			Reason: r.Reason,
		})
	}

	granted, newAppToken, reqID, err := s.orchestrator.RequestPermissions(ctx, claims.AppID, claims.UserID, coreRequests)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "permission request failed: %v", err)
	}

	return &pb.RequestPermissionsResponse{
		Granted:     granted,
		NewAppToken: newAppToken,
		RequestId:   reqID,
	}, nil
}

func (s *RPCServer) HasScopes(ctx context.Context, req *pb.HasScopesRequest) (*pb.HasScopesResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	grantedScopes, err := s.db.ListAppScopes(claims.AppID, claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query app scopes: %v", err)
	}

	grantedMap := make(map[string]bool)
	for _, sc := range grantedScopes {
		grantedMap[sc] = true
	}

	var missing []string
	for _, sc := range req.Scopes {
		if !grantedMap[sc] {
			missing = append(missing, sc)
		}
	}

	return &pb.HasScopesResponse{
		HasAll:        len(missing) == 0,
		MissingScopes: missing,
	}, nil
}

// ---------------------------------------------------------------------------
// IsAppInstalled
// ---------------------------------------------------------------------------

func (s *RPCServer) IsAppInstalled(ctx context.Context, req *pb.IsAppInstalledRequest) (*pb.IsAppInstalledResponse, error) {
	claims := claimsFromContext(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	if !hasScope(claims.Scopes, "contacts:read") {
		return nil, status.Error(codes.PermissionDenied, "missing scope: contacts:read")
	}

	if req.ContactId == "" || req.AppId == "" {
		return nil, status.Error(codes.InvalidArgument, "contact_id and app_id are required")
	}

	// Fetch contact from DB
	contact, err := s.db.GetContactByID(req.ContactId, claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database query error: %v", err)
	}
	if contact == nil {
		return nil, status.Errorf(codes.NotFound, "contact %q not found", req.ContactId)
	}

	// If the contact is a local contact (loopback), check local DB directly.
	isLocal := contact.PeerID == s.p2pService.HostID()
	if isLocal {
		app, err := s.db.GetAppByAppIDAndUserID(req.AppId, contact.TargetUserID)
		installed := (err == nil && app != nil)
		return &pb.IsAppInstalledResponse{Installed: installed}, nil
	}

	// Otherwise, generate request ID, send P2P message, and wait for response.
	requestID := uuid.NewString()
	ch := protocol.RegisterAppCheckRequest(requestID)
	defer protocol.DeregisterAppCheckRequest(requestID)

	payload := protocol.AppCheckRequestPayload{
		RequestID:    requestID,
		SenderUserID: claims.UserID,
		AppID:        req.AppId,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal payload: %v", err)
	}

	msg := &p2p.Message{
		AppID:        protocol.AppIDAppCheck,
		TargetUserID: contact.TargetUserID,
		Payload:      payloadBytes,
	}

	// Send message
	if err := s.p2pService.SendTo(ctx, contact.Multiaddr, contact.EncPubKey, msg); err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to send request to contact: %v", err)
	}

	// Wait for response or timeout (e.g. 5 seconds)
	select {
	case installed := <-ch:
		return &pb.IsAppInstalledResponse{Installed: installed}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, status.Error(codes.DeadlineExceeded, "timeout waiting for contact response")
	}
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
