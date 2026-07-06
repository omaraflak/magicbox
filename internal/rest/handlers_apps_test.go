package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magicbox/core/internal/core"
	"github.com/magicbox/core/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func TestListApps_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestListApps_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	// Insert mock app
	app := &db.App{
		ID:          "app-1",
		AppID:       "com.example.app",
		UserID:      "u1",
		Status:      "running",
		RouteSlug:   "app",
		Image:       "alpine",
		Version:     "1.0.0",
		ContainerID: "c1",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 || resp[0]["app_id"] != "com.example.app" {
		t.Errorf("unexpected list apps response: %+v", resp)
	}
}

func TestInstallApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Seed allowed registry for image verification
	_ = database.InsertRegistry("reg-1", "docker.io")

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	manifestJSON := `{
		"app_id": "com.example.test",
		"name": "Test App",
		"version": "1.0.0",
		"image": "docker.io/library/alpine:latest",
		"route_slug": "test",
		"entry_port": 80,
		"required_scopes": ["profile:read"]
	}`

	req := httptest.NewRequest("POST", "/api/v1/apps/install", bytes.NewReader([]byte(manifestJSON)))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["app_id"] != "com.example.test" {
		t.Errorf("expected app_id 'com.example.test', got: %v", resp["app_id"])
	}
}

func TestUninstallApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "running",
		RouteSlug: "app",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestStartApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	// Seed secret token to prevent rotate token failures inside o.Start
	_ = database.InsertAppToken("com.example.app", "u1", "secret-token")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "stopped",
		RouteSlug: "app",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/start", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestStopApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	app := &db.App{
		ID:          "app-1",
		AppID:       "com.example.app",
		UserID:      "u1",
		Status:      "running",
		ContainerID: "container-123",
		RouteSlug:   "app",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/stop", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestUpdateApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	// Seed app token for restart sequence
	_ = database.InsertAppToken("com.example.app", "u1", "secret-token")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "running",
		RouteSlug: "app",
	}
	_ = database.InsertApp(app)

	manifestJSON := `{
		"app_id": "com.example.app",
		"name": "Updated App",
		"version": "1.1.0",
		"image": "docker.io/library/alpine:latest",
		"route_slug": "app",
		"entry_port": 80,
		"required_scopes": ["profile:read"]
	}`

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/update", bytes.NewReader([]byte(manifestJSON)))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestRebuildApp_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	_ = database.InsertAppToken("com.example.app", "u1", "secret-token")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "running",
		RouteSlug: "app",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/rebuild", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestRotateToken_Success(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	_ = database.InsertAppToken("com.example.app", "u1", "secret-token")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "running",
		RouteSlug: "app",
	}
	_ = database.InsertApp(app)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/rotate-token", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestAppProxy_DockerNotInitialized(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "omar", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "omar", "pass")

	app := &db.App{
		ID:          "app-1",
		AppID:       "com.example.drive",
		UserID:      "u1",
		Status:      "running",
		ContainerID: "cont-1",
		RouteSlug:   "drive",
	}
	_ = database.InsertApp(app)

	// Route: /u/{username}/{routeSlug}/
	req := httptest.NewRequest("GET", "/u/omar/drive/", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Since setupTestServer sets docker client to nil, it should trigger our newly guarded check and return 500 error cleanly.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestDynamicPermissions_RestFlow(t *testing.T) {
	handler, database, _, orchestrator := setupTestServerWithOrch(t)


	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("u1", "user", string(hash), false)
	cookie := getSessionCookieForUser(t, handler, "user", "pass")

	app := &db.App{
		ID:        "app-1",
		AppID:     "com.example.app",
		UserID:    "u1",
		Status:    "running",
		RouteSlug: "app",
		Name:      "Example App",
	}
	_ = database.InsertApp(app)
	_ = database.InsertAppToken("com.example.app", "u1", "secret-token")


	// Verify no pending requests originally
	reqList := httptest.NewRequest("GET", "/api/v1/apps/permissions/requests", nil)
	reqList.AddCookie(cookie)
	rrList := httptest.NewRecorder()
	handler.ServeHTTP(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrList.Code)
	}
	if !strings.Contains(rrList.Body.String(), "[]") {
		t.Errorf("expected empty list, got %s", rrList.Body.String())
	}

	// Trigger a permission request synchronously
	granted, _, reqID, err := orchestrator.RequestPermissions(context.Background(), "com.example.app", "u1", []core.ScopeWithReason{
		{Scope: "contacts:read", Reason: "Reason details"},
	})
	if err != nil {
		t.Fatalf("RequestPermissions error: %v", err)
	}
	if granted {
		t.Error("expected granted to be false initially")
	}
	if reqID == "" {
		t.Error("expected non-empty request id")
	}

	// Verify it shows up in GET list
	reqList2 := httptest.NewRequest("GET", "/api/v1/apps/permissions/requests", nil)
	reqList2.AddCookie(cookie)
	rrList2 := httptest.NewRecorder()
	handler.ServeHTTP(rrList2, reqList2)

	if rrList2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrList2.Code)
	}
	if !strings.Contains(rrList2.Body.String(), `"app_id":"com.example.app"`) {
		t.Errorf("expected request in list, got %s", rrList2.Body.String())
	}

	// Approve it via POST /approve
	reqApprove := httptest.NewRequest("POST", "/api/v1/apps/permissions/requests/"+reqID+"/approve", nil)
	reqApprove.AddCookie(cookie)
	rrApprove := httptest.NewRecorder()
	handler.ServeHTTP(rrApprove, reqApprove)

	if rrApprove.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rrApprove.Code, rrApprove.Body.String())
	}

	// Verify scope is in DB after approval
	scopes, err := database.ListAppScopes("com.example.app", "u1")
	if err != nil {
		t.Fatalf("ListAppScopes failed: %v", err)
	}
	if len(scopes) != 1 || scopes[0] != "contacts:read" {
		t.Errorf("expected contacts:read scope to be granted, got %v", scopes)
	}
}

