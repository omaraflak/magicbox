package sdk

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	pb "github.com/magicbox/core/api/proto/v1"
	"google.golang.org/grpc"
)

func TestLoadEnv_Success(t *testing.T) {
	// Set all required env vars
	os.Setenv("MAGICBOX_API_TOKEN", "test-api-token")
	os.Setenv("MAGICBOX_CORE_URL", "test-core-url")
	os.Setenv("MAGICBOX_USER_ID", "test-user-id")
	os.Setenv("MAGICBOX_APP_ID", "test-app-id")
	os.Setenv("MAGICBOX_WEBHOOK_SECRET", "test-webhook-secret")
	defer func() {
		os.Unsetenv("MAGICBOX_API_TOKEN")
		os.Unsetenv("MAGICBOX_CORE_URL")
		os.Unsetenv("MAGICBOX_USER_ID")
		os.Unsetenv("MAGICBOX_APP_ID")
		os.Unsetenv("MAGICBOX_WEBHOOK_SECRET")
	}()

	env, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	if env.ApiToken != "test-api-token" {
		t.Errorf("expected ApiToken 'test-api-token', got %q", env.ApiToken)
	}
	if env.CoreURL != "test-core-url" {
		t.Errorf("expected CoreURL 'test-core-url', got %q", env.CoreURL)
	}
	if env.UserID != "test-user-id" {
		t.Errorf("expected UserID 'test-user-id', got %q", env.UserID)
	}
	if env.AppID != "test-app-id" {
		t.Errorf("expected AppID 'test-app-id', got %q", env.AppID)
	}
	if env.WebhookSecret != "test-webhook-secret" {
		t.Errorf("expected WebhookSecret 'test-webhook-secret', got %q", env.WebhookSecret)
	}
}

func TestLoadEnv_Missing(t *testing.T) {
	os.Setenv("MAGICBOX_API_TOKEN", "test-api-token")
	os.Setenv("MAGICBOX_CORE_URL", "test-core-url")
	os.Setenv("MAGICBOX_USER_ID", "test-user-id")
	os.Setenv("MAGICBOX_APP_ID", "test-app-id")
	os.Unsetenv("MAGICBOX_WEBHOOK_SECRET") // Missing!
	defer func() {
		os.Unsetenv("MAGICBOX_API_TOKEN")
		os.Unsetenv("MAGICBOX_CORE_URL")
		os.Unsetenv("MAGICBOX_USER_ID")
		os.Unsetenv("MAGICBOX_APP_ID")
	}()

	_, err := LoadEnv()
	if err == nil {
		t.Fatal("expected error due to missing MAGICBOX_WEBHOOK_SECRET, but got none")
	}
}

func TestVerifyWebhook(t *testing.T) {
	env := &Env{
		WebhookSecret: "super-secret-key",
	}

	tests := []struct {
		name          string
		headerValue   string
		expectSuccess bool
	}{
		{
			name:          "valid secret",
			headerValue:   "super-secret-key",
			expectSuccess: true,
		},
		{
			name:          "invalid secret",
			headerValue:   "wrong-key",
			expectSuccess: false,
		},
		{
			name:          "missing secret",
			headerValue:   "",
			expectSuccess: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tc.headerValue != "" {
				req.Header.Set("X-Magicbox-Webhook-Secret", tc.headerValue)
			}

			ok := env.VerifyWebhook(req)
			if ok != tc.expectSuccess {
				t.Errorf("VerifyWebhook() = %t; expected %t", ok, tc.expectSuccess)
			}
		})
	}
}

type mockMagicboxOSServer struct {
	pb.UnimplementedMagicboxOSServer
	hasScopesFunc         func(context.Context, *pb.HasScopesRequest) (*pb.HasScopesResponse, error)
	requestPermissionsFunc func(context.Context, *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error)
}

func (m *mockMagicboxOSServer) HasScopes(ctx context.Context, req *pb.HasScopesRequest) (*pb.HasScopesResponse, error) {
	if m.hasScopesFunc != nil {
		return m.hasScopesFunc(ctx, req)
	}
	return &pb.HasScopesResponse{HasAll: true}, nil
}

func (m *mockMagicboxOSServer) RequestPermissions(ctx context.Context, req *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error) {
	if m.requestPermissionsFunc != nil {
		return m.requestPermissionsFunc(ctx, req)
	}
	return &pb.RequestPermissionsResponse{Granted: true, NewAppToken: "new-token"}, nil
}

func TestEnsureScopes_AlreadyGranted(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	mockSrv := &mockMagicboxOSServer{
		hasScopesFunc: func(ctx context.Context, req *pb.HasScopesRequest) (*pb.HasScopesResponse, error) {
			return &pb.HasScopesResponse{HasAll: true}, nil
		},
	}
	pb.RegisterMagicboxOSServer(srv, mockSrv)
	go srv.Serve(lis)
	defer srv.Stop()

	env := &Env{
		ApiToken: "old-token",
		CoreURL:  "127.0.0.1:" + strings.Split(lis.Addr().String(), ":")[1],
		UserID:   "user-1",
		AppID:    "app-1",
	}

	err = env.EnsureScopes([]string{"scope-1"}, []string{"reason-1"})
	if err != nil {
		t.Fatalf("EnsureScopes failed: %v", err)
	}

	if env.ApiToken != "old-token" {
		t.Errorf("expected ApiToken to remain 'old-token', got %q", env.ApiToken)
	}
}

func TestEnsureScopes_NeedsRequest_ConsentRequired(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	mockSrv := &mockMagicboxOSServer{
		hasScopesFunc: func(ctx context.Context, req *pb.HasScopesRequest) (*pb.HasScopesResponse, error) {
			return &pb.HasScopesResponse{
				HasAll:        false,
				MissingScopes: []string{"scope-1"},
			}, nil
		},
		requestPermissionsFunc: func(ctx context.Context, req *pb.RequestPermissionsRequest) (*pb.RequestPermissionsResponse, error) {
			return &pb.RequestPermissionsResponse{
				Granted:   false,
				RequestId: "req-123",
			}, nil
		},
	}
	pb.RegisterMagicboxOSServer(srv, mockSrv)
	go srv.Serve(lis)
	defer srv.Stop()

	env := &Env{
		ApiToken: "old-token",
		CoreURL:  "127.0.0.1:" + strings.Split(lis.Addr().String(), ":")[1],
		UserID:   "user-1",
		AppID:    "app-1",
	}

	err = env.EnsureScopes([]string{"scope-1"}, []string{"reason-1"})
	if err == nil {
		t.Fatal("expected EnsureScopes to fail, got nil")
	}

	consentErr, ok := err.(*ConsentRequiredError)
	if !ok {
		t.Fatalf("expected ConsentRequiredError, got %T: %v", err, err)
	}

	if consentErr.RequestID != "req-123" {
		t.Errorf("expected RequestID 'req-123', got %q", consentErr.RequestID)
	}

	if len(consentErr.MissingScopes) != 1 || consentErr.MissingScopes[0] != "scope-1" {
		t.Errorf("expected MissingScopes ['scope-1'], got %v", consentErr.MissingScopes)
	}
}

