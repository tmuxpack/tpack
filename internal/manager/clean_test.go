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

	plugins := []plugin.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(plugins)

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
	plugins := []plugin.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(plugins)

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

	plugins := []plugin.Plugin{
		{Name: "tmux-sensible"},
	}

	mgr.Clean(plugins)

	// No remove messages.
	for _, msg := range output.OkMsgs {
		if len(msg) > 8 && msg[:8] == "Removing" {
			t.Error("no plugins should be removed")
		}
	}
}
