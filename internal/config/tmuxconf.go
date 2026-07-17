package config

import (
	"fmt"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// Collects all plugin definitions from:
// 1. Legacy @tpm_plugins tmux option
// 2. New @plugin syntax in tmux.conf + /etc/tmux.conf + sourced files (one level deep)
//
// Returns a non-nil error iff the resolved user tmux.conf (paths.TmuxConf)
// cannot be read. Resolve already fails closed on a nonexistent conf, so this
// fires only on exceptional read failures (permissions, TOCTOU deletion)
// between resolution and gathering. Reads of /etc/tmux.conf and of sourced
// files remain best-effort: conditional sourcing is legal, so their absence
// or unreadability is not an error.
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

	// New syntax: read config content.
	content, err := configContent(fs, paths.TmuxConf, paths.Home, paths.XDGConfigHome)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", paths.TmuxConf, err)
	}
	specs = append(specs, plug.ExtractPluginsFromConfig(content)...)

	// Parse all specs into Plugin structs.
	var plugins []plug.Plugin
	for _, raw := range specs {
		plugins = append(plugins, plug.ParseSpec(raw, warn))
	}
	return plugins, nil
}

// configContent reads /etc/tmux.conf + user tmux.conf + one level of sourced files.
// The user tmux.conf read failure propagates as an error; /etc/tmux.conf and
// sourced-file reads are best-effort and their failures are ignored.
func configContent(fs FS, tmuxConf, home, xdgConfigHome string) (string, error) {
	var b strings.Builder

	// /etc/tmux.conf (system config) -- best-effort.
	if data, err := fs.ReadFile("/etc/tmux.conf"); err == nil {
		b.Write(data)
	}

	// User tmux.conf -- must be readable.
	data, err := fs.ReadFile(tmuxConf)
	if err != nil {
		return "", err
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	b.Write(data)

	base := b.String()

	// Sourced files (one level deep, not recursive) -- best-effort.
	for _, file := range plug.ExtractSourcedFiles(base) {
		expanded := plug.ManualExpansion(file, home, xdgConfigHome)
		if data, err := fs.ReadFile(expanded); err == nil {
			b.WriteByte('\n')
			b.Write(data)
		}
	}

	return b.String(), nil
}
