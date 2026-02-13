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
// An optional "alias=X" token may follow the spec to override the plugin name.
// The branch suffix "#branch" may appear on either the spec or the alias token.
// Example: "catppuccin/tmux alias=catppuccin-tmux#v2"
func ParseSpec(raw string) Plugin {
	raw = strings.TrimSpace(raw)
	original := raw

	// Split on whitespace to find tokens.
	tokens := strings.Fields(raw)

	// Extract alias token if present.
	var alias string
	var specTokens []string
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "alias=") {
			alias = strings.TrimPrefix(tok, "alias=")
		} else {
			specTokens = append(specTokens, tok)
		}
	}

	// The remaining token is the spec (e.g. "catppuccin/tmux#v2").
	spec := ""
	if len(specTokens) > 0 {
		spec = specTokens[0]
	}

	// Extract branch from spec if present.
	var branch string
	if idx := strings.LastIndex(spec, "#"); idx > 0 {
		branch = spec[idx+1:]
		spec = spec[:idx]
	}

	// Extract branch from alias if present (and no branch found on spec).
	if alias != "" {
		if idx := strings.LastIndex(alias, "#"); idx > 0 {
			if branch == "" {
				branch = alias[idx+1:]
			}
			alias = alias[:idx]
		}
	}

	name := PluginName(spec)
	if alias != "" {
		name = alias
	}

	return Plugin{
		Raw:    original,
		Name:   name,
		Spec:   spec,
		Branch: branch,
		Alias:  alias,
	}
}
