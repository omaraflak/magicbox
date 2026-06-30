package main

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/sdk"
)

type mockCoreServer struct {
	pb.UnimplementedMagicboxOSServer
}

func (s *mockCoreServer) ListContacts(ctx context.Context, req *pb.ListContactsRequest) (*pb.ListContactsResponse, error) {
	return &pb.ListContactsResponse{
		Contacts: []*pb.Contact{
			{Id: "c1", DisplayName: "Alice"},
			{Id: "c2", DisplayName: "Bob"},
		},
	}, nil
}

func (s *mockCoreServer) SendToContact(ctx context.Context, req *pb.SendToContactRequest) (*pb.SendToContactResponse, error) {
	return &pb.SendToContactResponse{Success: true}, nil
}

func setupMockCoreServer(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMagicboxOSServer(grpcServer, &mockCoreServer{})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			// server closed
		}
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
	})

	coreURL = lis.Addr().String()
	apiToken = "test_token"
	env = &sdk.Env{
		CoreURL:  coreURL,
		ApiToken: apiToken,
		UserID:   "test_user",
		AppID:    "com.magicbox.drive",
	}
}
