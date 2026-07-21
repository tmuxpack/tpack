package config

import (
	"fmt"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// Collects all plugin definitions from:
// 1. Legacy @tpm_plugins tmux option
// 2. New @plugin syntax in tmux.conf + /etc/tmux.conf + recursively sourced files
//
// Returns a non-nil error if the resolved user tmux.conf or any required source
// cannot be read. /etc/tmux.conf and source-file -q directives are optional.
// TODO: Move to a separate config structure down the line, mayybe something akin to LazyVim
// warn, if non-nil, receives non-fatal parse warnings; a nil warn drops them.
func GatherPlugins(runner tmux.Runner, fs FS, paths Paths, warn func(string)) ([]plug.Plugin, error) {
	var specs []string

	if legacy, err := runner.ShowOption("@tpm_plugins"); err == nil && legacy != "" {
		for s := range strings.FieldsSeq(legacy) {
			s = strings.TrimSpace(s)
			if s != "" {
				specs = append(specs, s)
			}
		}
	}

	graph, err := LoadSourceGraph(fs, paths)
	if err != nil {
		return nil, err
	}
	for _, document := range graph.Documents() {
		specs = append(specs, plug.ExtractPluginsFromConfig(document.Content)...)
	}

	// Parse all specs into Plugin structs.
	var plugins []plug.Plugin
	for _, raw := range specs {
		plugin := plug.ParseSpec(raw, warn)
		if _, err := paths.PluginPath.Child(plugin.Name); err != nil {
			return nil, fmt.Errorf("invalid plugin %q: %w", raw, err)
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}
