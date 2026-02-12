package manager

import (
	"context"
	"strings"
	"sync"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

func (m *Manager) updateAll(ctx context.Context, plugins []plugin.Plugin) {
	m.output.Ok("Updating all plugins!")
	m.output.Ok("")

	var wg sync.WaitGroup
	for _, p := range plugins {
		if !m.IsPluginInstalled(p.Name) {
			continue
		}
		wg.Add(1)
		go func(p plugin.Plugin) {
			defer wg.Done()
			m.updatePlugin(ctx, p.Name)
		}(p)
	}
	wg.Wait()
}

func (m *Manager) updateSpecific(ctx context.Context, names []string) {
	var wg sync.WaitGroup
	for _, name := range names {
		pName := plugin.PluginName(name)
		if !m.IsPluginInstalled(pName) {
			m.output.Err(pName + " not installed!")
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			m.updatePlugin(ctx, name)
		}(pName)
	}
	wg.Wait()
}

func (m *Manager) updatePlugin(ctx context.Context, name string) {
	dir := plugin.PluginPath(name, m.pluginPath)
	output, err := m.puller.Pull(ctx, git.PullOptions{Dir: dir})

	indented := indentOutput(output)
	if err != nil {
		m.output.Err("  \"" + name + "\" update fail")
		m.output.Err(indented)
	} else {
		m.output.Ok("  \"" + name + "\" update success")
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
