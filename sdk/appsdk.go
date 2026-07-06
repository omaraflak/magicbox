package sdk

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Env holds the default Magicbox injected environment variables.
type Env struct {
	ApiToken      string
	CoreURL       string
	UserID        string
	AppID         string
	WebhookSecret string
}

// LoadEnv loads the injected environment variables.
func LoadEnv() (*Env, error) {
	apiToken := os.Getenv("MAGICBOX_API_TOKEN")
	coreURL := os.Getenv("MAGICBOX_CORE_URL")
	userID := os.Getenv("MAGICBOX_USER_ID")
	appID := os.Getenv("MAGICBOX_APP_ID")
	webhookSecret := os.Getenv("MAGICBOX_WEBHOOK_SECRET")

	if apiToken == "" || coreURL == "" || userID == "" || appID == "" || webhookSecret == "" {
		return nil, fmt.Errorf("missing one or more required Magicbox environment variables")
	}

	return &Env{
		ApiToken:      apiToken,
		CoreURL:       coreURL,
		UserID:        userID,
		AppID:         appID,
		WebhookSecret: webhookSecret,
	}, nil
}

// GetCoreClient dials the core host gRPC service and returns the authenticated client.
func (e *Env) GetCoreClient() (pb.MagicboxOSClient, *grpc.ClientConn, context.Context, error) {
	const maxMessageSize = 512 * 1024 * 1024 // 512 MB
	conn, err := grpc.Dial(
		e.CoreURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to dial core gRPC server: %w", err)
	}

	client := pb.NewMagicboxOSClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+e.ApiToken))

	return client, conn, ctx, nil
}

// HTMLHandler handles serving HTML frontend files (static assets + SPA routing)
// with dynamic <base href> tag injection.
type HTMLHandler struct {
	WebRoot string
}

// NewHTMLHandler creates a new HTMLHandler.
func NewHTMLHandler(webRoot string) *HTMLHandler {
	return &HTMLHandler{WebRoot: webRoot}
}

func (h *HTMLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try serving static asset directly
	cleanPath := filepath.Clean(r.URL.Path)
	filePath := filepath.Join(h.WebRoot, cleanPath)

	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}

	// Fallback to index.html with base tag injection
	indexPath := filepath.Join(h.WebRoot, "index.html")
	htmlBytes, err := os.ReadFile(indexPath)
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	basePath := "/"
	if prefix := r.Header.Get("X-Forwarded-Prefix"); prefix != "" {
		basePath = "/" + strings.Trim(prefix, "/") + "/"
	}

	baseTag := `<base href="` + basePath + `">`
	modified := strings.Replace(string(htmlBytes), "<head>", "<head>\n    "+baseTag, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(modified))
}

// WebhookMetadata extracts source details from webhook headers.
type WebhookMetadata struct {
	SourceApp  string
	SourceUser string
	SourceType string
}

// ParseWebhookMetadata extracts Magicbox webhook metadata from incoming request headers.
func ParseWebhookMetadata(r *http.Request) *WebhookMetadata {
	return &WebhookMetadata{
		SourceApp:  r.Header.Get("X-Magicbox-Source-App"),
		SourceUser: r.Header.Get("X-Magicbox-Source-User"),
		SourceType: r.Header.Get("X-Magicbox-Source-Type"),
	}
}

// VerifyWebhook checks if the incoming request has a valid X-Magicbox-Webhook-Secret header.
func (e *Env) VerifyWebhook(r *http.Request) bool {
	secret := r.Header.Get("X-Magicbox-Webhook-Secret")
	return secret != "" && secret == e.WebhookSecret
}

// EnsureScopes checks if the calling app is already granted a set of scopes,
// and if not, requests them dynamically from the core, blocking until user decision.
// If granted, it updates the ApiToken of the Env instance automatically.
func (e *Env) EnsureScopes(scopes []string, reasons []string) error {
	if len(scopes) != len(reasons) {
		return fmt.Errorf("ensure scopes: scopes and reasons length mismatch")
	}

	// Dial core and get client
	client, conn, ctx, err := e.GetCoreClient()
	if err != nil {
		return fmt.Errorf("ensure scopes: failed to dial core: %w", err)
	}
	defer conn.Close()

	// Check if already has scopes
	hasResp, err := client.HasScopes(ctx, &pb.HasScopesRequest{Scopes: scopes})
	if err != nil {
		return fmt.Errorf("ensure scopes: failed to check scopes: %w", err)
	}

	if hasResp.HasAll {
		return nil
	}

	// Request missing scopes
	var reqs []*pb.ScopeRequest
	missingMap := make(map[string]bool)
	for _, s := range hasResp.MissingScopes {
		missingMap[s] = true
	}

	for i, s := range scopes {
		if missingMap[s] {
			reqs = append(reqs, &pb.ScopeRequest{
				Scope:  s,
				Reason: reasons[i],
			})
		}
	}

	resp, err := client.RequestPermissions(ctx, &pb.RequestPermissionsRequest{
		Requests: reqs,
	})
	if err != nil {
		return fmt.Errorf("ensure scopes: failed to request permissions: %w", err)
	}

	if !resp.Granted {
		return fmt.Errorf("ensure scopes: permissions denied by user")
	}

	// Update local memory token so future calls use it
	if resp.NewAppToken != "" {
		e.ApiToken = resp.NewAppToken
	}

	return nil
}

