package manager

import (
	"context"
	"strings"
	"sync"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

const maxConcurrentUpdates = 5

func (m *Manager) updateAll(ctx context.Context, plugins []plugin.Plugin) {
	m.output.Ok("Updating all plugins!")
	m.output.Ok("")

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentUpdates)
	for _, p := range plugins {
		if !m.IsPluginInstalled(p.Name) {
			continue
		}
		wg.Add(1)
		go func(p plugin.Plugin) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			m.updatePlugin(ctx, p)
		}(p)
	}
	wg.Wait()
}

func (m *Manager) updateSpecific(ctx context.Context, plugins []plugin.Plugin, names []string) {
	// Build lookup map for branch info.
	pluginMap := make(map[string]plugin.Plugin)
	for _, p := range plugins {
		pluginMap[p.Name] = p
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentUpdates)
	for _, name := range names {
		pName := plugin.PluginName(name)
		if !m.IsPluginInstalled(pName) {
			m.output.Err(pName + " not installed!")
			continue
		}
		p := pluginMap[pName] // Get full plugin for branch info.
		if p.Name == "" {
			p = plugin.Plugin{Name: pName} // Fallback if not found in config.
		}
		wg.Add(1)
		go func(p plugin.Plugin) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			m.updatePlugin(ctx, p)
		}(p)
	}
	wg.Wait()
}

func (m *Manager) updatePlugin(ctx context.Context, p plugin.Plugin) {
	dir := plugin.PluginPath(p.Name, m.pluginPath)
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
