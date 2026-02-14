package manager

import (
	"os"

	"github.com/tmux-plugins/tpm/internal/plugin"
)

func (m *Manager) cleanPlugins(plugins []plugin.Plugin) {
	orphans := plugin.FindOrphans(plugins, m.pluginPath)
	for _, o := range orphans {
		m.output.Ok("Removing \"" + o.Name + "\"")
		if err := os.RemoveAll(o.Path); err != nil {
			m.output.Err("  \"" + o.Name + "\" clean fail")
		} else {
			if _, statErr := os.Stat(o.Path); statErr == nil {
				m.output.Err("  \"" + o.Name + "\" clean fail")
			} else {
				m.output.Ok("  \"" + o.Name + "\" clean success")
			}
		}
	}
}
