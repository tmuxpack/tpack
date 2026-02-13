package plugin

import (
	"path/filepath"
	"strings"
)

// PluginName extracts the plugin name from a raw specification.
// Examples:
//
//	"user/repo"                         → "repo"
//	"https://github.com/user/repo.git" → "repo"
//	"git@github.com:user/repo.git"     → "repo"
func PluginName(raw string) string {
	base := filepath.Base(raw)
	return strings.TrimSuffix(base, ".git")
}

// PluginPath returns the directory path for a plugin.
func PluginPath(raw, tpmPath string) string {
	return filepath.Join(tpmPath, PluginName(raw))
}

// NormalizeURL converts a shorthand plugin name to a full git URL.
// If the input already has a protocol prefix or contains ":", it is returned as-is.
// Otherwise it is expanded to a GitHub HTTPS URL.
// The "git::@" prefix is a credential placeholder used by the original TPM
// to prevent git from prompting for authentication on non-existent repos.
func NormalizeURL(shorthand string) string {
	if strings.Contains(shorthand, "://") || strings.Contains(shorthand, "git@") {
		return shorthand
	}
	return "https://git::@github.com/" + shorthand
}

// ParseSpec parses a raw plugin specification into a Plugin struct.
// The format is "spec#branch" where #branch is optional.
func ParseSpec(raw string) Plugin {
	raw = strings.TrimSpace(raw)
	var branch string
	if idx := strings.LastIndex(raw, "#"); idx > 0 {
		branch = raw[idx+1:]
		raw = raw[:idx]
	}
	return Plugin{
		Raw:    raw,
		Name:   PluginName(raw),
		Spec:   raw,
		Branch: branch,
	}
}
