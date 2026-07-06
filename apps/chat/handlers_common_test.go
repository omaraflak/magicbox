package main

import (
	"context"
	"net"
	"testing"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/sdk"
	"google.golang.org/grpc"
)

// Mock Core Server

type mockCoreServer struct {
	pb.UnimplementedMagicboxOSServer

	getProfileFunc         func(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	listContactsFunc       func(context.Context, *pb.ListContactsRequest) (*pb.ListContactsResponse, error)
	sendToContactFunc      func(context.Context, *pb.SendToContactRequest) (*pb.SendToContactResponse, error)
	sendContactRequestFunc func(context.Context, *pb.SendContactRequestRequest) (*pb.SendContactRequestResponse, error)
	getInviteLinkFunc      func(context.Context, *pb.GetInviteLinkRequest) (*pb.GetInviteLinkResponse, error)
	hasScopesFunc          func(context.Context, *pb.HasScopesRequest) (*pb.HasScopesResponse, error)
	requestPermissionsFunc func(context.Context, *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error)
}

func (m *mockCoreServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, req)
	}
	return &pb.GetProfileResponse{UserId: "test-user-id", Username: "test-user", CreatedAt: "2026-01-01T00:00:00Z"}, nil
}

func (m *mockCoreServer) ListContacts(ctx context.Context, req *pb.ListContactsRequest) (*pb.ListContactsResponse, error) {
	if m.listContactsFunc != nil {
		return m.listContactsFunc(ctx, req)
	}
	return &pb.ListContactsResponse{
		Contacts: []*pb.Contact{
			{Id: "c1", DisplayName: "Alice", TargetUserId: "alice-uid", InviteLink: "invite-alice"},
			{Id: "c2", DisplayName: "Bob", TargetUserId: "bob-uid", InviteLink: "invite-bob"},
		},
	}, nil
}

func (m *mockCoreServer) SendToContact(ctx context.Context, req *pb.SendToContactRequest) (*pb.SendToContactResponse, error) {
	if m.sendToContactFunc != nil {
		return m.sendToContactFunc(ctx, req)
	}
	return &pb.SendToContactResponse{Success: true}, nil
}

func (m *mockCoreServer) SendContactRequest(ctx context.Context, req *pb.SendContactRequestRequest) (*pb.SendContactRequestResponse, error) {
	if m.sendContactRequestFunc != nil {
		return m.sendContactRequestFunc(ctx, req)
	}
	return &pb.SendContactRequestResponse{Success: true, StatusMessage: "request sent"}, nil
}

func (m *mockCoreServer) GetInviteLink(ctx context.Context, req *pb.GetInviteLinkRequest) (*pb.GetInviteLinkResponse, error) {
	if m.getInviteLinkFunc != nil {
		return m.getInviteLinkFunc(ctx, req)
	}
	return &pb.GetInviteLinkResponse{InviteLink: "magicbox://invite/test"}, nil
}

func (m *mockCoreServer) HasScopes(ctx context.Context, req *pb.HasScopesRequest) (*pb.HasScopesResponse, error) {
	if m.hasScopesFunc != nil {
		return m.hasScopesFunc(ctx, req)
	}
	return &pb.HasScopesResponse{HasAll: true}, nil
}

func (m *mockCoreServer) RequestPermissions(ctx context.Context, req *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error) {
	if m.requestPermissionsFunc != nil {
		return m.requestPermissionsFunc(ctx, req)
	}
	return &pb.RequestPermissionsResponse{Granted: true}, nil
}

// Setup helpers

func setupTestEnvironment(t *testing.T) (*mockCoreServer, func()) {
	// Initialize in-memory SQLite DB
	db := setupTestDB(t)

	// Start local mock gRPC server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	mockServer := &mockCoreServer{}
	grpcServer := grpc.NewServer()
	pb.RegisterMagicboxOSServer(grpcServer, mockServer)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	env = &sdk.Env{
		CoreURL:       lis.Addr().String(),
		ApiToken:      "mock-api-token",
		UserID:        "test-user-id",
		AppID:         "com.magicbox.chat",
		WebhookSecret: "mock-webhook-secret",
	}

	cleanup := func() {
		grpcServer.Stop()
		lis.Close()
		db.Close()
		env = nil
	}

	return mockServer, cleanup
}
