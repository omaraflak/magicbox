package rpc

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/invite"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/rest"
)

// MockP2PService implements p2p.Service for testing.
type MockP2PService struct {
	hostID string
	sent   bool
}

func (m *MockP2PService) Start(ctx context.Context) error { return nil }
func (m *MockP2PService) Stop() error                     { return nil }
func (m *MockP2PService) HostID() string                  { return m.hostID }
func (m *MockP2PService) Multiaddrs() []string {
	return []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + m.hostID}
}
func (m *MockP2PService) RegisterHandler(appID string, handler p2p.Handler) {}
func (m *MockP2PService) SendTo(ctx context.Context, dest string, msg *p2p.Message) error {
	m.sent = true
	return nil
}

// makeInviteLink builds a properly encoded magicbox://invite/<base64> link for testing.
func makeInviteLink(multiaddr, userID, encPubKey string) string {
	payload := &invite.Payload{
		Multiaddr: multiaddr,
		UserID:    userID,
		EncPubKey: encPubKey,
	}
	link, _ := invite.Build(payload)
	return link
}

// setupGrpcTestServer initializes an in-memory gRPC server connection.
func setupGrpcTestServer(t *testing.T) (pb.MagicboxOSClient, *db.DB, *config.Config, *MockP2PService, func()) {
	tempDB := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(tempDB)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	logger, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := &config.Config{
		JWTSecret: []byte("jwt-secret"),
	}

	p2pMock := &MockP2PService{hostID: "local-host-id"}
	orch := core.NewOrchestrator(database, nil, cfg, logger, rest.GenerateAppToken)

	server := NewRPCServer(database, nil, orch, logger, cfg, p2pMock)

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer(grpc.UnaryInterceptor(server.authInterceptor))
	pb.RegisterMagicboxOSServer(s, server)
	go func() {
		s.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := pb.NewMagicboxOSClient(conn)

	cleanup := func() {
		conn.Close()
		s.GracefulStop()
		database.Close()
	}

	return client, database, cfg, p2pMock, cleanup
}

func TestGrpcGetProfile(t *testing.T) {
	client, database, _, _, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	// Seed user and app token
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret-1234567890")
	database.InsertAppScope("com.example.app", userID, "profile:read")

	token, _ := rest.GenerateAppToken([]byte("app-secret-1234567890"), userID, "com.example.app", []string{"profile:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	profileResp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profileResp.Username != "omar" || profileResp.UserId != userID {
		t.Errorf("unexpected profile response: %+v", profileResp)
	}
}

func TestGrpcListContacts(t *testing.T) {
	client, database, _, _, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	// Seed user, contacts, and app token
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")
	database.InsertAppScope("com.example.app", userID, "contacts:read")

	inviteLink := makeInviteLink("/ip4/1.2.3.4/tcp/4001/p2p/remote-peer-id", "alice-id", "test-enc-pub-key")
	database.AddContact("contact-1", userID, "Alice", inviteLink, "alice-id", "test-enc-pub-key")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{"contacts:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contactsResp.Contacts) != 1 || contactsResp.Contacts[0].DisplayName != "Alice" {
		t.Errorf("unexpected contacts response: %+v", contactsResp)
	}
}

func TestGrpcSendToContactRemote(t *testing.T) {
	client, database, _, p2pMock, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	// Seed user, contacts, and app token
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")

	inviteLink := makeInviteLink("/ip4/1.2.3.4/tcp/4001/p2p/remote-peer-id", "alice-id", "test-enc-pub-key")
	database.AddContact("contact-1", userID, "Alice", inviteLink, "alice-id", "test-enc-pub-key")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	sendResp, err := client.SendToContact(ctx, &pb.SendToContactRequest{
		ContactId: "contact-1",
		AppId:     "com.example.drive",
		Payload:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("SendToContact failed: %v", err)
	}
	if !sendResp.Success {
		t.Errorf("SendToContact response success is false: %s", sendResp.StatusMessage)
	}
	if !p2pMock.sent {
		t.Error("expected P2P service to be triggered for remote delivery")
	}
}

func TestGrpcSendToContactLoopback(t *testing.T) {
	client, database, _, p2pMock, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	// Seed user, contact matching local P2P host ID, and app token
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")

	// Multiaddress payload contains "local-host-id" to trigger loopback routing bypass
	inviteLink := makeInviteLink("/ip4/127.0.0.1/tcp/4001/p2p/local-host-id", userID, "test-enc-pub-key")
	database.AddContact("contact-1", userID, "Myself", inviteLink, userID, "test-enc-pub-key")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	// Note: local webhook dispatch will attempt a post request to the container IP.
	// Since no docker containers are running during this test, local routing will fail.
	// But we can check that it tried loopback (and did NOT call p2pMock.SendTo).
	_, err := client.SendToContact(ctx, &pb.SendToContactRequest{
		ContactId: "contact-1",
		AppId:     "com.example.drive",
		Payload:   []byte("hello"),
	})

	// It should attempt local loopback routing and fail because the container doesn't exist
	if err == nil {
		t.Error("expected local loopback to fail due to missing container IP")
	}
	if p2pMock.sent {
		t.Error("expected P2P SendTo NOT to be called for local loopback transfer")
	}
}
