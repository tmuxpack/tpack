// Package manager orchestrates tpack plugin operations.
package manager

import (
	"context"
	"os"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
)

// Coordinates plugin install, update, clean, and source operations.
type Manager struct {
	pluginPath string
	cloner     git.Cloner
	puller     git.Puller
	validator  git.Validator
	output     ui.Output
}

func New(pluginPath string, cloner git.Cloner, puller git.Puller, validator git.Validator, output ui.Output) *Manager {
	return &Manager{
		pluginPath: pluginPath,
		cloner:     cloner,
		puller:     puller,
		validator:  validator,
		output:     output,
	}
}

func (m *Manager) EnsurePathExists() error {
	return os.MkdirAll(m.pluginPath, 0o755)
}

// Checks if a plugin directory exists and is a git repo.
func (m *Manager) IsPluginInstalled(name string) bool {
	dir := plug.PluginPath(name, m.pluginPath)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	return m.validator.IsGitRepo(dir)
}

// Installs all listed plugins.
func (m *Manager) Install(ctx context.Context, plugins []plug.Plugin) {
	if err := m.EnsurePathExists(); err != nil {
		m.output.Err("Failed to create plugin directory: " + err.Error())
		return
	}
	m.verifyPathPermissions()
	for _, p := range plugins {
		m.installPlugin(ctx, p)
	}
}

// Updates the named plugins, or all if "all" is passed.
// TODO: an 'all' plugin name is hacky, needs a better way to specify all.
func (m *Manager) Update(ctx context.Context, plugins []plug.Plugin, names []string) {
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

// Removes plugin directories not in the list.
func (m *Manager) Clean(_ context.Context, plugins []plug.Plugin) {
	if err := m.EnsurePathExists(); err != nil {
		m.output.Err("Failed to create plugin directory: " + err.Error())
		return
	}
	m.cleanPlugins(plugins)
}
