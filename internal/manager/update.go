package manager

import (
	"context"
	"strings"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/parallel"
	"github.com/tmuxpack/tpack/internal/plug"
)

const maxConcurrentUpdates = 5

func (m *Manager) updateAll(ctx context.Context, plugins []plug.Plugin) {
	m.output.Ok("Updating all plugins!")
	m.output.Ok("")

	var installed []plug.Plugin
	for _, p := range plugins {
		if m.IsPluginInstalled(p.Name) {
			installed = append(installed, p)
		}
	}

	parallel.Do(installed, maxConcurrentUpdates, func(p plug.Plugin) {
		m.updatePlugin(ctx, p)
	})
}

func (m *Manager) updateSpecific(ctx context.Context, plugins []plug.Plugin, names []string) {
	// Build lookup map for branch info.
	pluginMap := make(map[string]plug.Plugin)
	for _, p := range plugins {
		pluginMap[p.Name] = p
	}

	var targets []plug.Plugin
	for _, name := range names {
		pName := plug.PluginName(name)
		if !m.IsPluginInstalled(pName) {
			m.output.Err(pName + " not installed!")
			continue
		}
		p := pluginMap[pName] // Get full plugin for branch info.
		if p.Name == "" {
			p = plug.Plugin{Name: pName} // Fallback if not found in config.
		}
		targets = append(targets, p)
	}

	parallel.Do(targets, maxConcurrentUpdates, func(p plug.Plugin) {
		m.updatePlugin(ctx, p)
	})
}

func (m *Manager) updatePlugin(ctx context.Context, p plug.Plugin) {
	dir := plug.PluginPath(p.Name, m.pluginPath)
	output, err := m.puller.Pull(ctx, git.PullOptions{Dir: dir, Branch: p.Branch})

	indented := indentOutput(output)
	if err != nil {
		m.output.Err("  \"" + p.Name + "\" update fail")
		m.output.Err(indented)
	} else {
		m.output.Ok("  \"" + p.Name + "\" update success")
		m.output.Ok(indented)
	}
}

func indentOutput(s string) string {
	if s == "" {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		lines = append(lines, "    | "+line)
	}
	return strings.Join(lines, "\n")
}
