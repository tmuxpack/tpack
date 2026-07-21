package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
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
	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin, nil),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Now update.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, updateOutput)
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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin, nil),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	// Update specific plugin.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, updateOutput)
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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

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
	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	declared := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin, nil),
	}
	installAll := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin, nil),
		plug.ParseSpec("tmux-plugins/tmux-sensible", nil),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, installAll)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with only tmux-example-plugin declared: tmux-sensible is now an orphan.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, cleanOutput)
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

func TestCleanWithEmptyConfigRemovesAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// Install a plugin; a readable-but-empty config declares nothing, so
	// clean is expected to remove it -- matching TPM's original contract.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	plugins := []plug.Plugin{
		plug.ParseSpec(tmuxExamplePlugin, nil),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.HasFailed() {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with an empty declared list (a readable config with zero
	// @plugin lines): everything installed is now an orphan and is removed.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, cleanOutput)
	mgr2.Clean(context.Background(), nil)

	dir := filepath.Join(pluginDir, "tmux-example-plugin")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected plugin to be removed when declared list is empty")
	}
}

func TestCleanAbortsWhenConfBecomesUnreadableAfterResolve(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test not supported on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping permission-denied test when running as root")
	}

	pluginDir, confFile := setupIntegrationDir(t)
	writeConf(t, confFile, `set -g @plugin "tmux-plugins/tmux-example-plugin"`)

	// An already-installed plugin that must survive: the command should
	// abort before ever reaching mgr.Clean.
	installedDir := filepath.Join(pluginDir, "tmux-example-plugin")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fs := config.RealFS{}
	paths := config.Paths{TmuxConf: confFile, Home: os.Getenv("HOME")}

	// The conf exists at Resolve time (mirrors config.Resolve's FileExists
	// check succeeding)...
	if !fs.FileExists(confFile) {
		t.Fatal("expected conf to exist before permission change")
	}

	// ...but becomes unreadable before GatherPlugins reads its content,
	// simulating a permission change or TOCTOU deletion race.
	if err := os.Chmod(confFile, 0o000); err != nil {
		t.Fatalf("failed to chmod conf file: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(confFile, 0o644) })

	_, err := config.GatherPlugins(&noopRunner{}, fs, paths, nil)
	if err == nil {
		t.Fatal("expected an explicit error when tmux.conf becomes unreadable after Resolve")
	}

	// Because GatherPlugins failed, a real command would exit non-zero
	// before calling mgr.Clean -- nothing on disk should be touched.
	if _, statErr := os.Stat(installedDir); statErr != nil {
		t.Errorf("expected installed plugin to survive an aborted clean, got: %v", statErr)
	}
}
