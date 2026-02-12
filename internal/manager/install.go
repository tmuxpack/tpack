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
	f.Close()
	os.Remove(f.Name())
}

func (m *Manager) installPlugin(ctx context.Context, p plugin.Plugin) {
	name := p.Name

	if m.IsPluginInstalled(name) {
		m.output.Ok("Already installed \"" + name + "\"")
		return
	}

	m.output.Ok("Installing \"" + name + "\"")

	dir := plugin.PluginPath(name, m.pluginPath)

	// Two-step clone: try raw URL first, then GitHub-expanded URL.
	err := m.cloner.Clone(ctx, git.CloneOptions{
		URL:    p.Spec,
		Dir:    dir,
		Branch: p.Branch,
	})
	if err != nil {
		// Fallback: expand to GitHub URL.
		ghURL := plugin.NormalizeURL(p.Spec)
		err = m.cloner.Clone(ctx, git.CloneOptions{
			URL:    ghURL,
			Dir:    dir,
			Branch: p.Branch,
		})
	}

	if err != nil {
		m.output.Err("  \"" + name + "\" download fail")
	} else {
		m.output.Ok("  \"" + name + "\" download success")
	}
}
