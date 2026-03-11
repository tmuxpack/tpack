package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
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

	plugins := []plug.Plugin{
		{Name: "tmux-test"},
	}

	mgr.Source(context.Background(), plugins)

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected *.tmux file to be executed")
	}
}

func TestSourceFallsBackToShOnBadShebang(t *testing.T) {
	pluginDir := setupTestDir(t)
	pDir := filepath.Join(pluginDir, "tmux-test")
	os.MkdirAll(pDir, 0o755)

	// Script with a broken shebang (simulates Termux where /usr/bin/env
	// does not exist). Direct exec fails with ErrNotFound; the sh fallback
	// should still run the script successfully.
	marker := filepath.Join(t.TempDir(), "sourced")
	script := filepath.Join(pDir, "test.tmux")
	os.WriteFile(script, []byte("#!/nonexistent/bin/sh\ntouch "+marker+"\n"), 0o755)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)
	mgr.Source(context.Background(), []plug.Plugin{{Name: "tmux-test"}})

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected sh fallback to execute script with broken shebang")
	}
	if output.HasFailed() {
		t.Errorf("expected no errors with sh fallback, got: %v", output.ErrMsgs)
	}
}

func TestSourceSkipsNonExistentDir(t *testing.T) {
	pluginDir := setupTestDir(t)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Name: "nonexistent"},
	}

	// Should not panic.
	mgr.Source(context.Background(), plugins)
}
