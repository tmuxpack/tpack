package manager

import (
	"os"

	"github.com/tmuxpack/tpack/internal/plug"
)

func (m *Manager) removeOrphans(orphans []plug.Orphan) {
	for _, o := range orphans {
		m.output.Ok("Removing \"" + o.Name + "\"")
		if err := os.RemoveAll(o.Path); err != nil {
			m.output.Err("  \"" + o.Name + "\" clean fail")
		} else {
			m.output.Ok("  \"" + o.Name + "\" clean success")
		}
	}
}
