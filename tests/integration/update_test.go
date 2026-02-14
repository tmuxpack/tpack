package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/plug"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestUpdateInstalledPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()

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

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
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

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
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

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()

	// Install a plugin, then clean with empty list.
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

	// Clean with empty plugin list (should remove the plugin).
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, cleanOutput)
	mgr2.Clean(nil)

	dir := filepath.Join(pluginDir, "tmux-example-plugin")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected plugin to be removed after clean")
	}
}
