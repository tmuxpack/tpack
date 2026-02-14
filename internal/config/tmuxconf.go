package config

import (
	"strings"

	"github.com/tmux-plugins/tpm/internal/plug"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// GatherPlugins collects all plugin definitions from:
// 1. Legacy @tpm_plugins tmux option
// 2. New @plugin syntax in tmux.conf + /etc/tmux.conf + sourced files (one level deep)
func GatherPlugins(runner tmux.Runner, fs FS, tmuxConf, home string) []plug.Plugin {
	var specs []string

	// Legacy: @tpm_plugins option.
	if legacy, err := runner.ShowOption("@tpm_plugins"); err == nil && legacy != "" {
		for _, s := range strings.Fields(legacy) {
			s = strings.TrimSpace(s)
			if s != "" {
				specs = append(specs, s)
			}
		}
	}

	// New syntax: read config content.
	content := configContent(fs, tmuxConf, home)
	specs = append(specs, plug.ExtractPluginsFromConfig(content)...)

	// Parse all specs into Plugin structs.
	var plugins []plug.Plugin
	for _, raw := range specs {
		plugins = append(plugins, plug.ParseSpec(raw))
	}
	return plugins
}

// configContent reads /etc/tmux.conf + user tmux.conf + one level of sourced files.
func configContent(fs FS, tmuxConf, home string) string {
	var parts []string

	// /etc/tmux.conf (system config)
	if data, err := fs.ReadFile("/etc/tmux.conf"); err == nil {
		parts = append(parts, string(data))
	}

	// User tmux.conf
	if data, err := fs.ReadFile(tmuxConf); err == nil {
		parts = append(parts, string(data))
	}

	base := strings.Join(parts, "\n")

	// Sourced files (one level deep, not recursive).
	var sourced []string
	for _, file := range plug.ExtractSourcedFiles(base) {
		expanded := plug.ManualExpansion(file, home)
		if data, err := fs.ReadFile(expanded); err == nil {
			sourced = append(sourced, string(data))
		}
	}

	if len(sourced) > 0 {
		return base + "\n" + strings.Join(sourced, "\n")
	}
	return base
}
