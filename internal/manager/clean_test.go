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

func TestCleanRemovesUnlisted(t *testing.T) {
	pluginDir := setupTestDir(t)
	// Install tmux-sensible (listed) and tmux-old (unlisted).
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")
	setupInstalledPlugin(t, pluginDir, "tmux-old")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(context.Background(), plugins)

	// tmux-old should be removed.
	if _, err := os.Stat(filepath.Join(pluginDir, "tmux-old")); !os.IsNotExist(err) {
		t.Error("tmux-old should have been removed")
	}

	// tmux-sensible should still exist.
	if _, err := os.Stat(filepath.Join(pluginDir, "tmux-sensible")); err != nil {
		t.Error("tmux-sensible should still exist")
	}

	foundRemove := false
	foundSuccess := false
	for _, msg := range output.OkMsgs {
		if msg == "Removing \"tmux-old\"" {
			foundRemove = true
		}
		if msg == "  \"tmux-old\" clean success" {
			foundSuccess = true
		}
	}
	if !foundRemove {
		t.Error("expected Removing message")
	}
	if !foundSuccess {
		t.Error("expected clean success message")
	}
}

func TestCleanNeverRemovesTpm(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tpm")
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	// Only list tmux-sensible (not tpm).
	plugins := []plug.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(context.Background(), plugins)

	// tpm should still exist.
	if _, err := os.Stat(filepath.Join(pluginDir, "tpm")); err != nil {
		t.Error("tpm directory should never be removed")
	}
}

func TestCleanNoPluginsToRemove(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(context.Background(), plugins)

	// No remove messages.
	for _, msg := range output.OkMsgs {
		if len(msg) > 8 && msg[:8] == "Removing" {
			t.Error("no plugins should be removed")
		}
	}
}
