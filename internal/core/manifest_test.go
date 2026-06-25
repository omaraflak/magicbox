package core

import (
	"testing"
)

func TestParseManifestDefaults(t *testing.T) {
	jsonData := `{
		"app_id": "com.example.test",
		"name": "Test App",
		"version": "1.0.0",
		"image": "docker.io/library/alpine:latest",
		"route_slug": "test"
	}`

	m, err := ParseManifest([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if m.WebhookPath != "/internal/magicbox-webhook" {
		t.Errorf("expected default WebhookPath '/internal/magicbox-webhook', got %q", m.WebhookPath)
	}

	if m.ResourceLimits == nil {
		t.Fatal("expected default ResourceLimits to be populated")
	}

	if m.ResourceLimits.MemoryMB != 256 {
		t.Errorf("expected default MemoryMB 256, got %d", m.ResourceLimits.MemoryMB)
	}

	if m.ResourceLimits.CPUCores != 0.5 {
		t.Errorf("expected default CPUCores 0.5, got %f", m.ResourceLimits.CPUCores)
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest Manifest
		valid    bool
	}{
		{
			name: "valid manifest",
			manifest: Manifest{
				AppID:          "com.example.app",
				Name:           "Valid App",
				Version:        "1.2.3",
				Image:          "docker.io/myrepo/myimage:tag",
				RouteSlug:      "my-app",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: true,
		},
		{
			name: "missing app_id",
			manifest: Manifest{
				Name:           "Invalid App",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: false,
		},
		{
			name: "invalid app_id format",
			manifest: Manifest{
				AppID:          "invalid_id",
				Name:           "Invalid App",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: false,
		},
		{
			name: "missing name",
			manifest: Manifest{
				AppID:          "com.example.app",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: false,
		},
		{
			name: "invalid version (not semver)",
			manifest: Manifest{
				AppID:          "com.example.app",
				Name:           "App",
				Version:        "v1.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: false,
		},
		{
			name: "reserved route slug",
			manifest: Manifest{
				AppID:          "com.example.app",
				Name:           "App",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "api",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"profile:read"},
			},
			valid: false,
		},
		{
			name: "invalid scope format",
			manifest: Manifest{
				AppID:          "com.example.app",
				Name:           "App",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"invalid_scope"},
			},
			valid: false,
		},
		{
			name: "valid shared scope",
			manifest: Manifest{
				AppID:          "com.example.app",
				Name:           "App",
				Version:        "1.0.0",
				Image:          "alpine",
				RouteSlug:      "slug",
				EntryPort:      9090,
				WebhookPath:    "/webhook",
				RequiredScopes: []string{"shared:photos:rw"},
			},
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateManifest(&tc.manifest)
			if tc.valid && len(errs) > 0 {
				t.Errorf("expected valid manifest, but got errors: %v", errs)
			}
			if !tc.valid && len(errs) == 0 {
				t.Error("expected validation errors, but got none")
			}
		})
	}
}

func TestScopeToVolumeAccess(t *testing.T) {
	tests := []struct {
		scope        string
		expectedName string
		expectedRO   bool
		expectedOK   bool
	}{
		{"shared:photos:rw", "photos", false, true},
		{"shared:documents:ro", "documents", true, true},
		{"profile:read", "", false, false},
		{"shared:photos", "", false, false},
		{"shared:photos:invalid", "", false, false},
	}

	for _, tc := range tests {
		name, ro, ok := ScopeToVolumeAccess(tc.scope)
		if name != tc.expectedName || ro != tc.expectedRO || ok != tc.expectedOK {
			t.Errorf("ScopeToVolumeAccess(%q) = (%q, %t, %t); expected (%q, %t, %t)",
				tc.scope, name, ro, ok, tc.expectedName, tc.expectedRO, tc.expectedOK)
		}
	}
}

func TestScopeToHumanReadable(t *testing.T) {
	tests := []struct {
		scope    string
		expected string
	}{
		{"profile:read", "Read your profile information"},
		{"contacts:read", "Read your contacts list"},
		{"shared:photos:rw", "Read and write your Photos"},
		{"shared:documents:ro", "Read your Documents"},
		{"invalid", "invalid"},
	}

	for _, tc := range tests {
		got := ScopeToHumanReadable(tc.scope)
		if got != tc.expected {
			t.Errorf("ScopeToHumanReadable(%q) = %q; expected %q", tc.scope, got, tc.expected)
		}
	}
}
