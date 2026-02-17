package manager

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tmuxpack/tpack/internal/plug"
)

// Source executes all *.tmux files from each plugin directory.
func (m *Manager) Source(ctx context.Context, plugins []plug.Plugin) {
	for _, p := range plugins {
		dir := plug.PluginPath(p.Name, m.pluginPath)
		m.sourcePlugin(ctx, dir)
	}
}

func (m *Manager) sourcePlugin(ctx context.Context, dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmux"))
	if err != nil {
		m.output.Err("glob error for " + dir + ": " + err.Error())
		return
	}

	for _, file := range matches {
		cmd := exec.CommandContext(ctx, file) //nolint:gosec // plugin files are user-configured
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			m.output.Err("error sourcing " + filepath.Base(file) + ": " + err.Error())
		}
	}
}
