// Package manager orchestrates TPM plugin operations.
package manager

import (
	"context"
	"os"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/ui"
)

// Manager coordinates plugin install, update, clean, and source operations.
type Manager struct {
	pluginPath string
	cloner     git.Cloner
	puller     git.Puller
	validator  git.Validator
	output     ui.Output
}

// New creates a Manager with the given dependencies.
func New(pluginPath string, cloner git.Cloner, puller git.Puller, validator git.Validator, output ui.Output) *Manager {
	return &Manager{
		pluginPath: pluginPath,
		cloner:     cloner,
		puller:     puller,
		validator:  validator,
		output:     output,
	}
}

// EnsurePathExists creates the plugin directory if it doesn't exist.
func (m *Manager) EnsurePathExists() error {
	return os.MkdirAll(m.pluginPath, 0o755)
}

// IsPluginInstalled checks if a plugin directory exists and is a git repo.
func (m *Manager) IsPluginInstalled(name string) bool {
	dir := plugin.PluginPath(name, m.pluginPath)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	return m.validator.IsGitRepo(dir)
}

// Install installs all listed plugins.
func (m *Manager) Install(ctx context.Context, plugins []plugin.Plugin) {
	if err := m.EnsurePathExists(); err != nil {
		m.output.Err("Failed to create plugin directory: " + err.Error())
		return
	}
	m.verifyPathPermissions()
	for _, p := range plugins {
		m.installPlugin(ctx, p)
	}
}

// Update updates the named plugins, or all if "all" is passed.
func (m *Manager) Update(ctx context.Context, plugins []plugin.Plugin, names []string) {
	if err := m.EnsurePathExists(); err != nil {
		m.output.Err("Failed to create plugin directory: " + err.Error())
		return
	}
	if len(names) == 1 && names[0] == "all" {
		m.updateAll(ctx, plugins)
		return
	}
	m.updateSpecific(ctx, plugins, names)
}

// Clean removes plugin directories not in the list.
func (m *Manager) Clean(plugins []plugin.Plugin) {
	if err := m.EnsurePathExists(); err != nil {
		m.output.Err("Failed to create plugin directory: " + err.Error())
		return
	}
	m.cleanPlugins(plugins)
}
