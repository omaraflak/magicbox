package rpc

import (
	"context"
	"encoding/json"
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
	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/invite"
	"github.com/magicbox/core/internal/keymanager"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/protocol"
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
func (m *MockP2PService) SendTo(ctx context.Context, peerMultiaddr string, encPubKeyHex string, msg *p2p.Message) error {
	m.sent = true
	if msg.AppID == protocol.AppIDAppCheck {
		var req protocol.AppCheckRequestPayload
		if err := json.Unmarshal(msg.Payload, &req); err == nil {
			go func() {
				protocol.ResolveAppCheckRequest(req.RequestID, true)
			}()
		}
	}
	return nil
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

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master identity key: %v", err)
	}
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}
	masterPubPEM, _ := crypto.MarshalPublicKey(masterPriv.Public())
	privPEM, _ := crypto.MarshalPublicKey(edPriv.Public()) // Public key for verifying contact reqs
	pubPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	encKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)
	encPubPEM, _ := crypto.MarshalPublicKey(xPriv.PublicKey())

	cfg := &config.Config{
		JWTSecret: []byte("jwt-secret"),
		Keys: &keymanager.KeyState{
			MasterPublicKeyPEM: masterPubPEM,
			PrivateKeyPEM:      privPEM,
			PublicKeyPEM:       pubPEM,
			EncryptionKeyPEM:   encKeyPEM,
			EncryptionPubPEM:   encPubPEM,
		},
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

func setupGrpcTestServerWithOrch(t *testing.T) (pb.MagicboxOSClient, *db.DB, *config.Config, *core.Orchestrator, func()) {
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

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master identity key: %v", err)
	}
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}
	masterPubPEM, _ := crypto.MarshalPublicKey(masterPriv.Public())
	privPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	pubPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	encKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)
	encPubPEM, _ := crypto.MarshalPublicKey(xPriv.PublicKey())

	cfg := &config.Config{
		JWTSecret: []byte("jwt-secret"),
		Keys: &keymanager.KeyState{
			MasterPublicKeyPEM: masterPubPEM,
			PrivateKeyPEM:      privPEM,
			PublicKeyPEM:       pubPEM,
			EncryptionKeyPEM:   encKeyPEM,
			EncryptionPubPEM:   encPubPEM,
		},
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

	return client, database, cfg, orch, cleanup
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

	database.AddContact("contact-1", userID, "Alice", "remote-peer-id", "/ip4/1.2.3.4/tcp/4001/p2p/remote-peer-id", "alice-id", "test-enc-pub-key", "alice-master-pub")

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

	database.AddContact("contact-1", userID, "Alice", "remote-peer-id", "/ip4/1.2.3.4/tcp/4001/p2p/remote-peer-id", "alice-id", "test-enc-pub-key", "alice-master-pub")

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

	// Multiaddress contains "local-host-id" and peerID matches to trigger loopback routing bypass
	database.AddContact("contact-1", userID, "Myself", "local-host-id", "/ip4/127.0.0.1/tcp/4001/p2p/local-host-id", userID, "test-enc-pub-key", "myself-master-pub")

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

func TestGrpcGetInviteLink(t *testing.T) {
	client, database, _, _, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")
	database.InsertAppScope("com.example.app", userID, "contacts:read")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{"contacts:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	resp, err := client.GetInviteLink(ctx, &pb.GetInviteLinkRequest{})
	if err != nil {
		t.Fatalf("GetInviteLink failed: %v", err)
	}

	if resp.InviteLink == "" {
		t.Fatal("expected non-empty invite link")
	}

	payload, err := invite.Parse(resp.InviteLink)
	if err != nil {
		t.Fatalf("failed to parse returned invite link: %v", err)
	}

	if payload.UserID != userID {
		t.Errorf("expected UserID %q, got %q", userID, payload.UserID)
	}
}

func TestGrpcSendContactRequest(t *testing.T) {
	client, database, _, _, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")
	database.InsertAppScope("com.example.app", userID, "contacts:write")

	// Generate a valid mock invite link targeting a remote peer
	mockInvite, _ := invite.Build(&invite.Payload{
		Multiaddr:    "/ip4/1.2.3.4/tcp/4001/p2p/12D3KooWRemotePeerID",
		UserID:       "remote-user-id",
		EncPubKey:    "remote-enc-pub-key-hex",
		MasterPubKey: "remote-master-pub-key-hex",
	})

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{"contacts:write"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	resp, err := client.SendContactRequest(ctx, &pb.SendContactRequestRequest{
		InviteLink:  mockInvite,
		DisplayName: "Bob",
	})
	if err != nil {
		t.Fatalf("SendContactRequest failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true, got false with message: %s", resp.StatusMessage)
	}

	// Verify request was written to DB
	req, err := database.GetContactRequestByTargetUserID(userID, "remote-user-id")
	if err != nil {
		t.Fatalf("failed to fetch contact request from db: %v", err)
	}
	if req == nil {
		t.Fatal("expected contact request to be found in database, got nil")
	}
	if req.DisplayName != "Bob" {
		t.Errorf("expected DisplayName 'Bob', got %q", req.DisplayName)
	}
}

func TestGrpcSendContactRequest_AutoAccept(t *testing.T) {
	client, database, _, _, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")
	database.InsertAppScope("com.example.app", userID, "contacts:write")

	// Insert an existing incoming contact request from remote-user-id to local userID
	err := database.InsertContactRequest(
		"req-id-123", userID, "incoming", "Bob (Incoming)",
		"12D3KooWRemotePeerID", "/ip4/1.2.3.4/tcp/4001/p2p/12D3KooWRemotePeerID", "remote-user-id", "remote-enc-pub-key-hex", "remote-master-pub-key-hex",
	)
	if err != nil {
		t.Fatalf("failed to insert incoming request: %v", err)
	}

	// Generate a valid mock invite link targeting the same remote user
	mockInvite, _ := invite.Build(&invite.Payload{
		Multiaddr:    "/ip4/1.2.3.4/tcp/4001/p2p/12D3KooWRemotePeerID",
		UserID:       "remote-user-id",
		EncPubKey:    "remote-enc-pub-key-hex",
		MasterPubKey: "remote-master-pub-key-hex",
	})

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{"contacts:write"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	resp, err := client.SendContactRequest(ctx, &pb.SendContactRequestRequest{
		InviteLink:  mockInvite,
		DisplayName: "Bob (Accepted)",
	})
	if err != nil {
		t.Fatalf("SendContactRequest failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true, got false with message: %s", resp.StatusMessage)
	}

	// Verify incoming request was deleted
	req, err := database.GetContactRequest(userID, "req-id-123")
	if err != nil {
		t.Fatalf("failed to check contact request: %v", err)
	}
	if req != nil {
		t.Error("expected incoming contact request to be deleted, but it still exists")
	}

	// Verify contact was created
	c, err := database.GetContactByTargetUserID(userID, "remote-user-id")
	if err != nil {
		t.Fatalf("failed to query contact from db: %v", err)
	}
	if c == nil {
		t.Fatal("expected contact to be created in database, got nil")
	}
	if c.DisplayName != "Bob (Incoming)" {
		t.Errorf("expected DisplayName 'Bob (Incoming)' (from the incoming request), got %q", c.DisplayName)
	}
}

func TestGrpcRequestPermissions_Approve(t *testing.T) {
	client, database, _, orch, cleanup := setupGrpcTestServerWithOrch(t)
	defer cleanup()

	userID := "user-123"
	appID := "com.example.app"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertApp(&db.App{
		ID:          "app-id-123",
		AppID:       appID,
		Name:        "Example App",
		Image:       "image",
		Version:     "1.0.0",
		RouteSlug:   "slug",
		Host:        "host",
		UserID:      userID,
		Status:      "running",
		ContainerID: "container-id",
	})
	database.InsertAppToken(appID, userID, "app-secret")
	database.InsertAppScope(appID, userID, "profile:read")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, appID, []string{"profile:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	resp, err := client.RequestPermissions(ctx, &pb.RequestPermissionsRequest{
		Requests: []*pb.ScopeRequest{
			{Scope: "contacts:read", Reason: "need contacts"},
		},
	})
	if err != nil {
		t.Fatalf("RequestPermissions failed: %v", err)
	}

	if resp.Granted {
		t.Error("expected permissions not to be granted immediately")
	}

	if resp.RequestId == "" {
		t.Error("expected a request ID to be returned")
	}

	// Verify it is in the pending permissions list
	reqs := orch.ListPendingPermissions(userID)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 pending permission request, got %d", len(reqs))
	}

	// Approve it using the RequestID
	ok := orch.ApprovePermissionRequest(context.Background(), resp.RequestId, nil)
	if !ok {
		t.Fatal("expected ApprovePermissionRequest to succeed")
	}

	// Verify that contacts:read is now in DB scopes
	scopes, err := database.ListAppScopes(appID, userID)
	if err != nil {
		t.Fatalf("ListAppScopes failed: %v", err)
	}
	hasContactsRead := false
	for _, s := range scopes {
		if s == "contacts:read" {
			hasContactsRead = true
			break
		}
	}
	if !hasContactsRead {
		t.Error("expected contacts:read to be added to database app scopes")
	}
}

func TestGrpcRequestPermissions_Reject(t *testing.T) {
	client, database, _, orch, cleanup := setupGrpcTestServerWithOrch(t)
	defer cleanup()

	userID := "user-123"
	appID := "com.example.app"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertApp(&db.App{
		ID:          "app-id-123",
		AppID:       appID,
		Name:        "Example App",
		Image:       "image",
		Version:     "1.0.0",
		RouteSlug:   "slug",
		Host:        "host",
		UserID:      userID,
		Status:      "running",
		ContainerID: "container-id",
	})
	database.InsertAppToken(appID, userID, "app-secret")
	database.InsertAppScope(appID, userID, "profile:read")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, appID, []string{"profile:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	resp, err := client.RequestPermissions(ctx, &pb.RequestPermissionsRequest{
		Requests: []*pb.ScopeRequest{
			{Scope: "contacts:write", Reason: "need write"},
		},
	})
	if err != nil {
		t.Fatalf("RequestPermissions failed: %v", err)
	}

	if resp.Granted {
		t.Error("expected permissions not to be granted immediately")
	}

	// Reject the request using the RequestID
	ok := orch.RejectPermissionRequest(resp.RequestId)
	if !ok {
		t.Fatal("expected RejectPermissionRequest to succeed")
	}

	// Verify it is removed from the pending list
	reqs := orch.ListPendingPermissions(userID)
	if len(reqs) != 0 {
		t.Errorf("expected 0 pending requests, got %d", len(reqs))
	}
}

func TestGrpcIsAppInstalled(t *testing.T) {
	client, database, _, p2pMock, cleanup := setupGrpcTestServer(t)
	defer cleanup()

	// Seed user, contacts, and app token
	userID := "user-123"
	database.CreateUser(userID, "omar", "hash", false)
	database.InsertAppToken("com.example.app", userID, "app-secret")
	database.InsertAppScope("com.example.app", userID, "contacts:read")

	// 1. Add a remote contact
	database.AddContact("contact-remote", userID, "Alice", "remote-peer-id", "/ip4/1.2.3.4/tcp/4001/p2p/remote-peer-id", "alice-id", "test-enc-pub-key", "alice-master-pub")

	// 2. Add a local contact (loopback)
	database.AddContact("contact-local", userID, "Bob", p2pMock.HostID(), "/ip4/127.0.0.1/tcp/4001/p2p/"+p2pMock.HostID(), "bob-id", "test-enc-pub-key-2", "bob-master-pub")

	token, _ := rest.GenerateAppToken([]byte("app-secret"), userID, "com.example.app", []string{"contacts:read"})
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	// Test case A: local (loopback) contact where app is NOT installed
	resp, err := client.IsAppInstalled(ctx, &pb.IsAppInstalledRequest{
		ContactId: "contact-local",
		AppId:     "com.example.drive",
	})
	if err != nil {
		t.Fatalf("IsAppInstalled local failed: %v", err)
	}
	if resp.Installed {
		t.Error("expected app not to be installed for local contact")
	}

	// Test case B: local (loopback) contact where app IS installed
	database.CreateUser("bob-id", "bob", "hash", false)
	if err := database.InsertApp(&db.App{
		ID:        "app-drive",
		AppID:     "com.example.drive",
		Name:      "Drive",
		UserID:    "bob-id",
		Status:    "installed",
		RouteSlug: "drive",
		Image:     "drive-image",
	}); err != nil {
		t.Fatalf("InsertApp failed: %v", err)
	}
	resp, err = client.IsAppInstalled(ctx, &pb.IsAppInstalledRequest{
		ContactId: "contact-local",
		AppId:     "com.example.drive",
	})
	if err != nil {
		t.Fatalf("IsAppInstalled local failed: %v", err)
	}
	if !resp.Installed {
		t.Error("expected app to be installed for local contact")
	}

	// Test case C: remote contact
	resp, err = client.IsAppInstalled(ctx, &pb.IsAppInstalledRequest{
		ContactId: "contact-remote",
		AppId:     "com.example.chat",
	})
	if err != nil {
		t.Fatalf("IsAppInstalled remote failed: %v", err)
	}
	if !resp.Installed {
		t.Error("expected app to be installed for remote contact")
	}
	if !p2pMock.sent {
		t.Error("expected P2P message to be sent")
	}
}
