package manager

import (
	"context"
	"os"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

func (m *Manager) verifyPathPermissions() {
	// Probe actual write access by attempting to create a temp file.
	f, err := os.CreateTemp(m.pluginPath, ".tpm-probe-*")
	if err != nil {
		m.output.Err(m.pluginPath + " is not writable!")
		return
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
}

func (m *Manager) installPlugin(ctx context.Context, p plugin.Plugin) {
	name := p.Name

	if m.IsPluginInstalled(name) {
		m.output.Ok("Already installed \"" + name + "\"")
		return
	}

	m.output.Ok("Installing \"" + name + "\"")

	dir := plugin.PluginPath(name, m.pluginPath)

	err := git.CloneWithFallback(ctx, m.cloner, git.CloneOptions{
		URL:    p.Spec,
		Dir:    dir,
		Branch: p.Branch,
	}, plugin.NormalizeURL)

	if err != nil {
		m.output.Err("  \"" + name + "\" download fail")
	} else {
		m.output.Ok("  \"" + name + "\" download success")
	}
}
