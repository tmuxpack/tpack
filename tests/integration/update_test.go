package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
)

func TestUpdateInstalledPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// First install.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(pluginDir, cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Now update.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, updateOutput)
	mgr2.Update(ctx, plugins, []string{"all"})

	if updateOutput.HasFailed() {
		t.Errorf("update reported failure: %v", updateOutput.ErrMsgs)
	}

	// Should have "update success" message.
	found := false
	for _, msg := range updateOutput.OkMsgs {
		if msg == "  \"tmux-example-plugin\" update success" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected update success message, got: %v", updateOutput.OkMsgs)
	}
}

func TestUpdateSpecificPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()
	installOutput := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	// Update specific plugin.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, updateOutput)
	mgr2.Update(ctx, plugins, []string{"tmux-example-plugin"})

	if updateOutput.HasFailed() {
		t.Errorf("update reported failure: %v", updateOutput.ErrMsgs)
	}
}

func TestUpdateNotInstalledPlugin(t *testing.T) {
	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	mgr.Update(context.Background(), nil, []string{"nonexistent"})

	found := false
	for _, msg := range output.ErrMsgs {
		if msg == "nonexistent not installed!" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'not installed' error, got: %v", output.ErrMsgs)
	}
}

func TestCleanRemovesUnlistedPlugins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// Install a declared plugin plus an unlisted orphan.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(pluginDir, cloner, puller, validator, installOutput)
	declared := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin),
	}
	installAll := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin),
		plug.ParseSpec("tmux-plugins/tmux-sensible"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, installAll)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with only tmux-example-plugin declared: tmux-sensible is now an orphan.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, cleanOutput)
	mgr2.Clean(context.Background(), declared)

	orphanDir := filepath.Join(pluginDir, "tmux-sensible")
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Error("expected unlisted plugin to be removed after clean")
	}

	declaredDir := filepath.Join(pluginDir, "tmux-example-plugin")
	if _, err := os.Stat(declaredDir); err != nil {
		t.Error("expected declared plugin to survive clean")
	}
}

func TestCleanWithEmptyPluginListRemovesNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// Install a plugin so there is something on disk that must NOT be removed.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(pluginDir, cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with an empty declared list (e.g. a missing or unreadable
	// tmux.conf): must never treat every installed plugin as an orphan.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, cleanOutput)
	mgr2.Clean(context.Background(), nil)

	dir := filepath.Join(pluginDir, "tmux-example-plugin")
	if _, err := os.Stat(dir); err != nil {
		t.Error("expected plugin to survive clean when declared list is empty")
	}
}
