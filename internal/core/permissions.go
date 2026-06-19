package core

import (
	"strings"
)

// ScopeToHumanReadable converts a machine scope string into a user-friendly description.
//   - "profile:read"       -> "Read your profile information"
//   - "shared:photos:rw"   -> "Read and write your Photos"
//   - "shared:photos:ro"   -> "Read your Photos"
func ScopeToHumanReadable(scope string) string {
	if scope == "profile:read" {
		return "Read your profile information"
	}

	parts := strings.Split(scope, ":")
	if len(parts) != 3 || parts[0] != "shared" {
		return scope
	}

	name := capitalize(parts[1])
	switch parts[2] {
	case "rw":
		return "Read and write your " + name
	case "ro":
		return "Read your " + name
	default:
		return scope
	}
}

// ValidateScopes returns true only if every scope in requested is present in granted.
func ValidateScopes(requested []string, granted []string) bool {
	grantedSet := make(map[string]bool, len(granted))
	for _, s := range granted {
		grantedSet[s] = true
	}
	for _, s := range requested {
		if !grantedSet[s] {
			return false
		}
	}
	return true
}

// ScopeToVolumeAccess parses a shared scope into its volume name and access mode.
//   - "shared:photos:rw" -> ("photos", false, true)
//   - "shared:photos:ro" -> ("photos", true, true)
//   - Non-shared scopes  -> ("", false, false)
func ScopeToVolumeAccess(scope string) (name string, readOnly bool, ok bool) {
	parts := strings.Split(scope, ":")
	if len(parts) != 3 || parts[0] != "shared" {
		return "", false, false
	}

	switch parts[2] {
	case "ro":
		return parts[1], true, true
	case "rw":
		return parts[1], false, true
	default:
		return "", false, false
	}
}

// capitalize returns s with its first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
