package plugin

import (
	"regexp"
	"strings"
)

var (
	// Matches: set -g @plugin "...", set-option -g @plugin '...'
	// with optional leading whitespace.
	pluginLineRe = regexp.MustCompile(
		`^[ \t]*set(?:-option)?\s+-g\s+@plugin\s+['"]?([^'"]+)['"]?`)

	// Matches: source "...", source-file -q "...", source '...'
	sourcedFileRe = regexp.MustCompile(
		`^[ \t]*source(?:-file)?\s+(?:-q\s+)?['"]?([^'"]+)['"]?`)
)

// ExtractPluginsFromConfig parses tmux config content and returns all
// plugin specifications found in @plugin declarations.
func ExtractPluginsFromConfig(content string) []string {
	var plugins []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if m := pluginLineRe.FindStringSubmatch(line); m != nil {
			spec := strings.TrimSpace(m[1])
			if spec != "" {
				plugins = append(plugins, spec)
			}
		}
	}
	return plugins
}

// ExtractSourcedFiles parses tmux config content and returns all
// file paths referenced by source or source-file commands.
func ExtractSourcedFiles(content string) []string {
	var files []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if m := sourcedFileRe.FindStringSubmatch(line); m != nil {
			path := strings.TrimSpace(m[1])
			if path != "" {
				files = append(files, path)
			}
		}
	}
	return files
}

// ManualExpansion expands ~ and $HOME in a path, mirroring the bash behavior.
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
