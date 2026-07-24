package plug

import (
	"fmt"
	"strings"
)

// NormalizeURL converts a shorthand plugin name to a full git URL.
// If the input already has a protocol prefix or contains ":", it is returned as-is.
// Otherwise it is expanded to a GitHub HTTPS URL.
// The "git::@" prefix is a credential placeholder used by the original TPM
// to prevent git from prompting for authentication on non-existent repos.
func NormalizeURL(shorthand string) string {
	if strings.Contains(shorthand, "://") {
		return shorthand
	}
	if _, _, ok := parseSCPIdentity(shorthand); ok {
		return shorthand
	}
	return "https://git::@github.com/" + shorthand
}

// ParseSpec parses a raw plugin specification into a Plugin struct.
// The format is "spec#branch" where #branch is optional.
// An optional "alias=X" token may follow the spec to override DirName.
// The branch suffix "#branch" may appear on either the spec or the alias token.
// Example: "catppuccin/tmux alias=catppuccin-tmux#v2"
// warn, if non-nil, receives a message for unexpected extra tokens; a nil
// warn silently drops it.
func ParseSpec(raw string, warn func(string)) (Plugin, error) {
	if containsControl(raw) || strings.ContainsAny(raw, `\'";`+"`$|&<>") {
		return Plugin{}, fmt.Errorf("parse plugin %q: repository spec contains unsafe characters", raw)
	}
	raw = strings.TrimSpace(raw)
	original := raw

	// Split on whitespace to find tokens.
	tokens := strings.Fields(raw)

	// Extract alias token if present.
	var alias string
	var specTokens []string
	for _, tok := range tokens {
		if after, ok := strings.CutPrefix(tok, "alias="); ok {
			alias = after
		} else {
			specTokens = append(specTokens, tok)
		}
	}

	// The remaining token is the spec (e.g. "catppuccin/tmux#v2").
	spec := ""
	if len(specTokens) > 0 {
		spec = specTokens[0]
	}
	if len(specTokens) > 1 && warn != nil {
		warn(fmt.Sprintf("plugin spec %q has unexpected extra tokens: %s",
			raw, strings.Join(specTokens[1:], " ")))
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

	identity, err := NormalizeIdentity(spec)
	if err != nil {
		return Plugin{}, fmt.Errorf("parse plugin %q: %w", original, err)
	}
	dirName := GeneratedDirName(identity)
	if alias != "" {
		dirName = alias
	}

	return Plugin{
		Raw: original, Name: RepositoryName(identity), Identity: identity, DirName: dirName,
		Spec: spec, Branch: branch, Alias: alias,
	}, nil
}
