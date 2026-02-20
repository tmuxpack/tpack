package plug

import (
	"regexp"
	"strings"
)

var (
	// Matches: set -g @plugin "...", set-option -g @plugin '...',
	// or unquoted set -g @plugin value, with optional leading whitespace.
	// Three alternations handle double-quoted, single-quoted, and unquoted values.
	pluginLineRe = regexp.MustCompile(
		`^[ \t]*set(?:-option)?\s+-g\s+@plugin\s+(?:"([^"]+)"|'([^']+)'|(\S+))`)

	// Matches: source "...", source-file -q "...", source '...', or unquoted path.
	// Three alternations handle double-quoted, single-quoted, and unquoted values.
	sourcedFileRe = regexp.MustCompile(
		`^[ \t]*source(?:-file)?\s+(?:-q\s+)?(?:"([^"]+)"|'([^']+)'|(\S+))`)
)

// extractMatches scans content line by line and collects the first
// non-empty capture group from re for each non-comment line that matches.
func extractMatches(content string, re *regexp.Regexp) []string {
	var results []string
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if m := re.FindStringSubmatch(line); m != nil {
			// m[1] = double-quoted, m[2] = single-quoted, m[3] = unquoted
			val := m[1]
			if val == "" {
				val = m[2]
			}
			if val == "" {
				val = m[3]
			}
			val = strings.TrimSpace(val)
			if val != "" {
				results = append(results, val)
			}
		}
	}
	return results
}

// MatchesPluginLine reports whether line is a @plugin declaration for the given spec.
func MatchesPluginLine(line, spec string) bool {
	m := pluginLineRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	val := m[1]
	if val == "" {
		val = m[2]
	}
	if val == "" {
		val = m[3]
	}
	return strings.TrimSpace(val) == spec
}

// ExtractPluginsFromConfig parses tmux config content and returns all
// plugin specifications found in @plugin declarations.
func ExtractPluginsFromConfig(content string) []string {
	return extractMatches(content, pluginLineRe)
}

// ExtractSourcedFiles parses tmux config content and returns all
// file paths referenced by source or source-file commands.
func ExtractSourcedFiles(content string) []string {
	return extractMatches(content, sourcedFileRe)
}

// ManualExpansion expands ~ and $HOME in a path, mirroring the bash behavior.
// TODO: Should add support for ${HOME}
func ManualExpansion(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return home + path[1:]
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "$HOME/") {
		return home + path[5:]
	}
	if path == "$HOME" {
		return home
	}
	return path
}
