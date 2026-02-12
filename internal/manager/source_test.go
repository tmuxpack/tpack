package manager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestSourceExecutesTmuxFiles(t *testing.T) {
	pluginDir := setupTestDir(t)
	pDir := filepath.Join(pluginDir, "tmux-test")
	os.MkdirAll(pDir, 0o755)

	// Create a *.tmux file that touches a marker file.
	marker := filepath.Join(t.TempDir(), "sourced")
	script := filepath.Join(pDir, "test.tmux")
	os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		{Name: "tmux-test"},
	}

	mgr.Source(plugins)

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected *.tmux file to be executed")
	}
}

func TestSourceSkipsNonExistentDir(t *testing.T) {
	pluginDir := setupTestDir(t)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		{Name: "nonexistent"},
	}

	// Should not panic.
	mgr.Source(plugins)
}
