// Package core provides domain types and validation for Magicbox applications.
package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Manifest describes the metadata and requirements for a Magicbox application.
type Manifest struct {
	AppID          string          `json:"app_id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Image          string          `json:"image"`
	EntryPort      int             `json:"entry_port"`
	RouteSlug      string          `json:"route_slug"`
	WebhookPath    string          `json:"webhook_path"`
	Host           string          `json:"host"`
	RequiredScopes []string        `json:"required_scopes"`
	VolumeMounts   []VolumeMount   `json:"volume_mounts"`
	ResourceLimits *ResourceLimits `json:"resource_limits"`
}

// VolumeMount describes a volume to be mounted into the application container.
type VolumeMount struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Access string `json:"access"`
}

// ResourceLimits specifies the resource constraints for a container.
type ResourceLimits struct {
	MemoryMB int     `json:"memory_mb"`
	CPUCores float64 `json:"cpu_cores"`
}

// Compiled regexes for manifest validation.
var (
	appIDRegex     = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*){2,}$`)
	semverRegex    = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	routeSlugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}[a-z0-9]$`)
	scopeRegex     = regexp.MustCompile(`^(profile:read|contacts:read|shared:[a-z][a-z0-9-]*:(ro|rw))$`)
	volumeNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	hostRegex       = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)
)

// reservedSlugs contains route slugs that are reserved for internal use.
var reservedSlugs = map[string]bool{
	"api":    true,
	"admin":  true,
	"setup":  true,
	"auth":   true,
	"static": true,
	"health": true,
	"u":      true,
}

// ParseManifest parses JSON data into a Manifest and applies default values.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}

	// Apply defaults.
	if m.WebhookPath == "" {
		m.WebhookPath = "/internal/magicbox-webhook"
	}
	if m.ResourceLimits == nil {
		m.ResourceLimits = &ResourceLimits{
			MemoryMB: 256,
			CPUCores: 0.5,
		}
	}

	return &m, nil
}

// ValidateManifest checks a manifest against all validation rules and returns
// a list of human-readable error strings. An empty slice means the manifest is valid.
func ValidateManifest(m *Manifest) []string {
	var errs []string

	// app_id: required, reverse-DNS with at least 3 segments.
	if m.AppID == "" {
		errs = append(errs, "app_id is required")
	} else if !appIDRegex.MatchString(m.AppID) {
		errs = append(errs, "app_id must match format 'segment.segment.segment' (lowercase alphanumeric, at least 3 segments)")
	}

	// name: required, 1-64 chars.
	if m.Name == "" {
		errs = append(errs, "name is required")
	} else if len(m.Name) > 64 {
		errs = append(errs, "name must be 1-64 characters")
	}

	// version: required, semver.
	if m.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverRegex.MatchString(m.Version) {
		errs = append(errs, "version must be valid semver (e.g. 1.0.0)")
	}

	// image: required.
	if m.Image == "" {
		errs = append(errs, "image is required")
	}

	// entry_port: 1-65535.
	if m.EntryPort < 1 || m.EntryPort > 65535 {
		errs = append(errs, "entry_port must be between 1 and 65535")
	}

	// route_slug: required, pattern, not reserved.
	if m.RouteSlug == "" {
		errs = append(errs, "route_slug is required")
	} else if !routeSlugRegex.MatchString(m.RouteSlug) {
		errs = append(errs, "route_slug must be 2-32 lowercase alphanumeric characters or hyphens, starting and ending with alphanumeric")
	} else if reservedSlugs[m.RouteSlug] {
		errs = append(errs, fmt.Sprintf("route_slug %q is reserved", m.RouteSlug))
	}

	// host: optional, valid hostname.
	if m.Host != "" && !hostRegex.MatchString(m.Host) {
		errs = append(errs, "host must be a valid domain name or hostname (e.g. magic.box)")
	}

	// webhook_path: must start with "/".
	if !strings.HasPrefix(m.WebhookPath, "/") {
		errs = append(errs, "webhook_path must start with /")
	}

	// required_scopes: non-empty, each must match pattern.
	if len(m.RequiredScopes) == 0 {
		errs = append(errs, "required_scopes must contain at least one scope")
	} else {
		for _, scope := range m.RequiredScopes {
			if !scopeRegex.MatchString(scope) {
				errs = append(errs, fmt.Sprintf("invalid scope %q", scope))
			}
		}
	}

	// Build set of granted shared volume names from scopes for cross-validation.
	scopeVolumes := make(map[string]bool)
	for _, scope := range m.RequiredScopes {
		parts := strings.Split(scope, ":")
		if len(parts) == 3 && parts[0] == "shared" {
			scopeVolumes[parts[1]] = true
		}
	}

	// volume_mounts validation.
	for i, vol := range m.VolumeMounts {
		if vol.Type != "shared" {
			errs = append(errs, fmt.Sprintf("volume_mounts[%d].type must be \"shared\"", i))
		}
		if !volumeNameRegex.MatchString(vol.Name) {
			errs = append(errs, fmt.Sprintf("volume_mounts[%d].name must be lowercase alphanumeric with hyphens", i))
		}
		if vol.Access != "read-only" && vol.Access != "read-write" {
			errs = append(errs, fmt.Sprintf("volume_mounts[%d].access must be \"read-only\" or \"read-write\"", i))
		}
		if !scopeVolumes[vol.Name] {
			errs = append(errs, fmt.Sprintf("volume_mounts[%d].name %q has no corresponding scope in required_scopes", i, vol.Name))
		}
	}

	// resource_limits validation.
	if m.ResourceLimits != nil {
		if m.ResourceLimits.MemoryMB < 1 || m.ResourceLimits.MemoryMB > 4096 {
			errs = append(errs, "resource_limits.memory_mb must be between 1 and 4096")
		}
		if m.ResourceLimits.CPUCores < 0.1 || m.ResourceLimits.CPUCores > 4.0 {
			errs = append(errs, "resource_limits.cpu_cores must be between 0.1 and 4.0")
		}
	}

	return errs
}
