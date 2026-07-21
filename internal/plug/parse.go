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
	// Captures the optional quiet flag and double-quoted, single-quoted, or unquoted paths.
	sourcedFileRe = regexp.MustCompile(
		`^[ \t]*source(?:-file)?\s+(-q\s+)?(?:"([^"]+)"|'([^']+)'|(\S+))`)
)

const (
	quietGroup      = 1
	pathDoubleGroup = 2
	pathSingleGroup = 3
	pathBareGroup   = 4
)

// SourceDirective describes a source command and whether read failure is allowed.
type SourceDirective struct {
	Path     string
	Optional bool
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

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

// ExtractSourceDirectives parses source and source-file commands from tmux config content.
func ExtractSourceDirectives(content string) []SourceDirective {
	var directives []SourceDirective
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		m := sourcedFileRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		path := firstNonEmpty(m[pathDoubleGroup], m[pathSingleGroup], m[pathBareGroup])
		if path != "" {
			directives = append(directives, SourceDirective{
				Path:     strings.TrimSpace(path),
				Optional: m[quietGroup] != "",
			})
		}
	}
	return directives
}

// ManualExpansion expands ~, $HOME, ${HOME}, $XDG_CONFIG_HOME, and
// ${XDG_CONFIG_HOME} in a path. When xdgConfigHome is empty, XDG
// variables are left unexpanded.
func ManualExpansion(path, home, xdgConfigHome string) string {
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
	if strings.HasPrefix(path, "${HOME}/") {
		return home + path[7:]
	}
	if path == "${HOME}" {
		return home
	}
	if xdgConfigHome != "" {
		if strings.HasPrefix(path, "$XDG_CONFIG_HOME/") {
			return xdgConfigHome + path[16:]
		}
		if path == "$XDG_CONFIG_HOME" {
			return xdgConfigHome
		}
		if strings.HasPrefix(path, "${XDG_CONFIG_HOME}/") {
			return xdgConfigHome + path[18:]
		}
		if path == "${XDG_CONFIG_HOME}" {
			return xdgConfigHome
		}
	}
	return path
}
