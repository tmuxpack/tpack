package config

import (
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// Collects all plugin definitions from:
// 1. Legacy @tpm_plugins tmux option
// 2. New @plugin syntax in tmux.conf + /etc/tmux.conf + sourced files (one level deep)
// TODO: Move to a separate config structure down the line, mayybe something akin to LazyVim
func GatherPlugins(runner tmux.Runner, fs FS, tmuxConf, home string) []plug.Plugin {
	var specs []string

	if legacy, err := runner.ShowOption("@tpm_plugins"); err == nil && legacy != "" {
		for s := range strings.FieldsSeq(legacy) {
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
	var b strings.Builder

	// /etc/tmux.conf (system config)
	if data, err := fs.ReadFile("/etc/tmux.conf"); err == nil {
		b.Write(data)
	}

	// User tmux.conf
	if data, err := fs.ReadFile(tmuxConf); err == nil {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.Write(data)
	}

	base := b.String()

	// Sourced files (one level deep, not recursive).
	for _, file := range plug.ExtractSourcedFiles(base) {
		expanded := plug.ManualExpansion(file, home)
		if data, err := fs.ReadFile(expanded); err == nil {
			b.WriteByte('\n')
			b.Write(data)
		}
	}

	return b.String()
}
