package manager

import (
	"context"
	"os"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

func (m *Manager) verifyPathPermissions() {
	pluginPath, err := m.pluginRoot.Path()
	if err != nil {
		m.output.Err(err.Error())
		return
	}
	// Probe actual write access by attempting to create a temp file.
	f, err := os.CreateTemp(pluginPath, ".tpack-probe-*")
	if err != nil {
		m.output.Err(pluginPath + " is not writable!")
		return
	}
	_ = f.Close()
	_ = os.Remove(f.Name()) //nolint:gosec // path from os.CreateTemp is safe
}

func (m *Manager) installPlugin(ctx context.Context, p plug.Plugin) {
	name := p.Name

	if m.IsPluginInstalled(p.DirName) {
		m.output.Ok("Already installed \"" + name + "\"")
		return
	}

	m.output.Ok("Installing \"" + name + "\"")

	dir, err := m.pluginRoot.Child(p.DirName)
	if err != nil {
		m.output.Err("invalid plugin path for " + name + ": " + err.Error())
		return
	}

	err = git.CloneWithFallback(ctx, m.cloner, git.CloneOptions{
		URL:    p.Spec,
		Dir:    dir,
		Branch: p.Branch,
		OnWarning: func(msg string) {
			m.output.Warn("  \"" + name + "\" warning: " + msg)
		},
	}, plug.NormalizeURL)

	if err != nil {
		m.output.Err("  \"" + name + "\" download fail")
	} else {
		m.output.Ok("  \"" + name + "\" download success")
	}
}
