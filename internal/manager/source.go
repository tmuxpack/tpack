package manager

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tmux-plugins/tpm/internal/plugin"
)

// Source executes all *.tmux files from each plugin directory.
func (m *Manager) Source(plugins []plugin.Plugin) {
	for _, p := range plugins {
		dir := plugin.PluginPath(p.Name, m.pluginPath)
		m.sourcePlugin(dir)
	}
}

func (m *Manager) sourcePlugin(dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmux"))
	if err != nil {
		return
	}

	for _, file := range matches {
		cmd := exec.Command(file) //nolint:gosec,noctx // plugin files are user-configured, no cancellation needed
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		_ = cmd.Run()
	}
}
