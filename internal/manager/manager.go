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
	pluginRoot plug.Root
	cloner     git.Cloner
	puller     git.Puller
	validator  git.Validator
	output     ui.Output
}

func New(pluginRoot plug.Root, cloner git.Cloner, puller git.Puller, validator git.Validator, output ui.Output) *Manager {
	return &Manager{
		pluginRoot: pluginRoot,
		cloner:     cloner,
		puller:     puller,
		validator:  validator,
		output:     output,
	}
}

func (m *Manager) EnsurePathExists() error {
	pluginPath, err := m.pluginRoot.Path()
	if err != nil {
		return err
	}
	return os.MkdirAll(pluginPath, 0o755)
}

// Checks if a plugin directory exists and is a git repo.
func (m *Manager) IsPluginInstalled(name string) bool {
	dir, err := m.pluginRoot.Child(name)
	if err != nil {
		return false
	}
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
	resolved, err := m.pluginRoot.Resolved()
	if err != nil {
		m.output.Err("unsafe plugin directory: " + err.Error())
		return
	}
	orphans, err := plug.FindOrphans(plugins, resolved)
	if err != nil {
		m.output.Err("failed to inspect plugin directory: " + err.Error())
		return
	}
	m.removeOrphans(orphans)
}
