package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/magicbox/core/internal/docker"
	"golang.org/x/crypto/bcrypt"
)

type MockDockerClient struct {
	LocalImageDigestsFn     func(ctx context.Context, img string) ([]string, error)
	RemoteImageDigestFn     func(ctx context.Context, img string) (string, error)
	InspectContainerFn      func(ctx context.Context, containerID string) (*docker.ContainerStatus, error)
	InspectRawContainerFn   func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	PullImageFn             func(ctx context.Context, img string, force bool) (string, error)
	RenameContainerFn       func(ctx context.Context, containerID, newName string) error
	CreateCoreContainerFn   func(ctx context.Context, newImage string, old *types.ContainerJSON) (string, error)
	StartUpdaterContainerFn func(ctx context.Context, oldName, newName string) error
	RemoveContainerFn       func(ctx context.Context, containerID string) error
}

func (m *MockDockerClient) LocalImageDigests(ctx context.Context, img string) ([]string, error) {
	if m.LocalImageDigestsFn != nil {
		return m.LocalImageDigestsFn(ctx, img)
	}
	return nil, nil
}

func (m *MockDockerClient) RemoteImageDigest(ctx context.Context, img string) (string, error) {
	if m.RemoteImageDigestFn != nil {
		return m.RemoteImageDigestFn(ctx, img)
	}
	return "", nil
}

func (m *MockDockerClient) InspectContainer(ctx context.Context, containerID string) (*docker.ContainerStatus, error) {
	if m.InspectContainerFn != nil {
		return m.InspectContainerFn(ctx, containerID)
	}
	return nil, nil
}

func (m *MockDockerClient) InspectRawContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.InspectRawContainerFn != nil {
		return m.InspectRawContainerFn(ctx, containerID)
	}
	return types.ContainerJSON{}, nil
}

func (m *MockDockerClient) PullImage(ctx context.Context, img string, force bool) (string, error) {
	if m.PullImageFn != nil {
		return m.PullImageFn(ctx, img, force)
	}
	return "", nil
}

func (m *MockDockerClient) RenameContainer(ctx context.Context, containerID, newName string) error {
	if m.RenameContainerFn != nil {
		return m.RenameContainerFn(ctx, containerID, newName)
	}
	return nil
}

func (m *MockDockerClient) CreateCoreContainer(ctx context.Context, newImage string, old *types.ContainerJSON) (string, error) {
	if m.CreateCoreContainerFn != nil {
		return m.CreateCoreContainerFn(ctx, newImage, old)
	}
	return "", nil
}

func (m *MockDockerClient) StartUpdaterContainer(ctx context.Context, oldName, newName string) error {
	if m.StartUpdaterContainerFn != nil {
		return m.StartUpdaterContainerFn(ctx, oldName, newName)
	}
	return nil
}

func (m *MockDockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	if m.RemoveContainerFn != nil {
		return m.RemoveContainerFn(ctx, containerID)
	}
	return nil
}

func TestCheckUpdates_Unauthenticated(t *testing.T) {
	handler, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/updates/check", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestCheckUpdates_Admin(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("admin-id", "admin", string(hash), true)

	// Login admin to get session cookie
	adminCookie := getSessionCookieForUser(t, handler, "admin", "pass")

	req := httptest.NewRequest("GET", "/api/v1/updates/check", nil)
	req.AddCookie(adminCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp CheckUpdatesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Core == nil {
		t.Error("expected Core update info for admin user, got nil")
	} else {
		if resp.Core.Image != "docker.io/omaraflak/magicbox-core:latest" {
			t.Errorf("expected core image docker.io/omaraflak/magicbox-core:latest, got %q", resp.Core.Image)
		}
		if resp.Core.Error != "docker client not initialized (mock)" {
			t.Errorf("expected mock docker client error, got %q", resp.Core.Error)
		}
	}
}

func TestCheckUpdates_StandardUser(t *testing.T) {
	handler, database, _ := setupTestServer(t)

	// Create standard user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	_ = database.CreateUser("user-id", "user", string(hash), false)

	// Login standard user
	userCookie := getSessionCookieForUser(t, handler, "user", "pass")

	req := httptest.NewRequest("GET", "/api/v1/updates/check", nil)
	req.AddCookie(userCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp CheckUpdatesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Core != nil {
		t.Errorf("expected Core update info to be nil for standard user, got %+v", resp.Core)
	}
}

func TestCheckImageUpdate_NilDocker(t *testing.T) {
	s := &Server{
		docker: nil,
	}
	_, _, _, err := s.checkImageUpdate(context.Background(), "ref", "id", "")
	if err == nil || err.Error() != "docker client not initialized" {
		t.Errorf("expected docker client not initialized error, got %v", err)
	}
}

func TestCheckImageUpdate_NoUpdateAvailable(t *testing.T) {
	mockDocker := &MockDockerClient{
		LocalImageDigestsFn: func(ctx context.Context, img string) ([]string, error) {
			if img == "local-id" {
				return []string{"repo@sha256:matched-digest"}, nil
			}
			return nil, nil
		},
		RemoteImageDigestFn: func(ctx context.Context, img string) (string, error) {
			return "sha256:matched-digest", nil
		},
	}

	s := &Server{
		docker: mockDocker,
	}

	local, latest, available, err := s.checkImageUpdate(context.Background(), "myref:latest", "local-id", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if available {
		t.Error("expected updateAvailable to be false, got true")
	}
	if len(local) != 1 || local[0] != "repo@sha256:matched-digest" {
		t.Errorf("unexpected local digests: %v", local)
	}
	if latest != "sha256:matched-digest" {
		t.Errorf("unexpected latest digest: %s", latest)
	}
}

func TestCheckImageUpdate_UpdateAvailable(t *testing.T) {
	mockDocker := &MockDockerClient{
		LocalImageDigestsFn: func(ctx context.Context, img string) ([]string, error) {
			if img == "local-id" {
				return []string{"repo@sha256:old-digest"}, nil
			}
			return nil, nil
		},
		RemoteImageDigestFn: func(ctx context.Context, img string) (string, error) {
			return "sha256:new-digest", nil
		},
	}

	s := &Server{
		docker: mockDocker,
	}

	_, latest, available, err := s.checkImageUpdate(context.Background(), "myref:latest", "local-id", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Error("expected updateAvailable to be true, got false")
	}
	if latest != "sha256:new-digest" {
		t.Errorf("unexpected latest digest: %s", latest)
	}
}

func TestCheckImageUpdate_FallbackToImageRef(t *testing.T) {
	mockDocker := &MockDockerClient{
		LocalImageDigestsFn: func(ctx context.Context, img string) ([]string, error) {
			if img == "myref:latest" {
				return []string{"repo@sha256:ref-digest"}, nil
			}
			return nil, fmt.Errorf("id lookup failed")
		},
		RemoteImageDigestFn: func(ctx context.Context, img string) (string, error) {
			return "sha256:new-digest", nil
		},
	}

	s := &Server{
		docker: mockDocker,
	}

	local, _, available, err := s.checkImageUpdate(context.Background(), "myref:latest", "local-id", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Error("expected updateAvailable to be true")
	}
	if len(local) != 1 || local[0] != "repo@sha256:ref-digest" {
		t.Errorf("expected local digests to fall back to imageRef lookup, got %v", local)
	}
}

func TestCheckImageUpdate_FallbackToDigest(t *testing.T) {
	mockDocker := &MockDockerClient{
		LocalImageDigestsFn: func(ctx context.Context, img string) ([]string, error) {
			return nil, fmt.Errorf("not found")
		},
		RemoteImageDigestFn: func(ctx context.Context, img string) (string, error) {
			return "sha256:new-digest", nil
		},
	}

	s := &Server{
		docker: mockDocker,
	}

	local, _, _, err := s.checkImageUpdate(context.Background(), "myref:latest", "local-id", "fallback-digest-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(local) != 1 || local[0] != "fallback-digest-value" {
		t.Errorf("expected local digests to fall back to fallbackDigest parameter, got %v", local)
	}
}
