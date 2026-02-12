package manager

import (
	"os"
	"path/filepath"

	"github.com/tmux-plugins/tpm/internal/plugin"
)

func (m *Manager) cleanPlugins(plugins []plugin.Plugin) {
	// Build exact set of plugin names for lookup.
	nameSet := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		nameSet[p.Name] = true
	}

	entries, err := os.ReadDir(m.pluginPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := plugin.PluginName(entry.Name())

		// Never remove the "tpm" directory itself.
		if name == "tpm" {
			continue
		}

		// Check if this directory matches any listed plugin.
		if nameSet[name] {
			continue
		}

		m.output.Ok("Removing \"" + name + "\"")
		dir := filepath.Join(m.pluginPath, entry.Name())
		if err := os.RemoveAll(dir); err != nil {
			m.output.Err("  \"" + name + "\" clean fail")
		} else {
			// Verify removal.
			if _, statErr := os.Stat(dir); statErr == nil {
				m.output.Err("  \"" + name + "\" clean fail")
			} else {
				m.output.Ok("  \"" + name + "\" clean success")
			}
		}
	}
}
