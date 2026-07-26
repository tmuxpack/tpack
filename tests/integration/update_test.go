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
		t.Skip("skipping Git integration test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// First install.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	plugins := createLocalPlugins(t, tmuxExamplePlugin)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.Result() != nil {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Now update.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, updateOutput)
	mgr2.Update(ctx, plugins, []string{"all"})

	if updateOutput.Result() != nil {
		t.Errorf("update reported failure: %v", updateOutput.ErrMsgs)
	}

	// Should have "update success" message.
	found := false
	for _, msg := range updateOutput.OkMsgs {
		if msg == "  \"tmux-plugins/tmux-example-plugin\" update success" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected update success message, got: %v", updateOutput.OkMsgs)
	}
}

func TestUpdateSpecificPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Git integration test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()
	installOutput := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	plugins := createLocalPlugins(t, tmuxExamplePlugin)
	p := plugins[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	// Update specific plugin.
	updateOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, updateOutput)
	mgr2.Update(ctx, plugins, []string{p.Name})

	if updateOutput.Result() != nil {
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
	p := mustParsePlugin(t, "example/nonexistent")

	mgr.Update(context.Background(), []plug.Plugin{p}, []string{p.Name})

	found := false
	for _, msg := range output.ErrMsgs {
		if msg == "example/nonexistent not installed!" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'not installed' error, got: %v", output.ErrMsgs)
	}
}

func TestCleanRemovesUnlistedPlugins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Git integration test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := gitcli.NewCloner()
	puller := gitcli.NewPuller()
	validator := gitcli.NewValidator()

	// Install a declared plugin plus an unlisted orphan.
	installOutput := ui.NewMockOutput()
	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, installOutput)
	installAll := createLocalPlugins(t, tmuxExamplePlugin, "tmux-plugins/tmux-sensible")
	declaredPlugin := installAll[0]
	orphanPlugin := installAll[1]
	declared := []plug.Plugin{declaredPlugin}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, installAll)

	if installOutput.Result() != nil {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with only tmux-example-plugin declared: tmux-sensible is now an orphan.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, cleanOutput)
	mgr2.Clean(context.Background(), declared)

	orphanDir := pluginPath(t, pluginDir, orphanPlugin)
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Error("expected unlisted plugin to be removed after clean")
	}

	declaredDir := pluginPath(t, pluginDir, declaredPlugin)
	if _, err := os.Stat(declaredDir); err != nil {
		t.Error("expected declared plugin to survive clean")
	}
}

func TestCleanWithEmptyConfigRemovesAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Git integration test in -short mode")
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
	plugins := createLocalPlugins(t, tmuxExamplePlugin)
	p := plugins[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if installOutput.Result() != nil {
		t.Fatalf("install failed: %v", installOutput.ErrMsgs)
	}

	// Clean with an empty declared list (a readable config with zero
	// @plugin lines): everything installed is now an orphan and is removed.
	cleanOutput := ui.NewMockOutput()
	mgr2 := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, cleanOutput)
	mgr2.Clean(context.Background(), nil)

	dir := pluginPath(t, pluginDir, p)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected plugin to be removed when declared list is empty")
	}
}

func TestCleanPreservesPluginsFromEveryConfigRoot(t *testing.T) {
	pluginDir, firstConf := setupIntegrationDir(t)
	secondConf := filepath.Join(filepath.Dir(firstConf), "second.conf")
	firstPlugin := mustParsePlugin(t, tmuxExamplePlugin)
	secondPlugin := mustParsePlugin(t, "tmux-plugins/tmux-sensible")

	writeConf(t, firstConf, `set -g @plugin "`+firstPlugin.Raw+`"`)
	writeConf(t, secondConf, `set -g @plugin "`+secondPlugin.Raw+`"`)
	for _, plugin := range []plug.Plugin{firstPlugin, secondPlugin} {
		if err := os.MkdirAll(pluginPath(t, pluginDir, plugin), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	paths := config.Paths{
		TmuxConf:   secondConf,
		TmuxConfs:  []string{firstConf, secondConf},
		PluginPath: mustRoot(t, pluginDir),
		Home:       os.Getenv("HOME"),
	}
	plugins, err := config.GatherPlugins(&noopRunner{}, config.RealFS{}, paths, nil)
	if err != nil {
		t.Fatal(err)
	}

	output := ui.NewMockOutput()
	mgr := manager.New(paths.PluginPath, nil, nil, nil, output)
	mgr.Clean(context.Background(), plugins)
	if output.Result() != nil {
		t.Fatalf("clean failed: %v", output.ErrMsgs)
	}

	for _, plugin := range []plug.Plugin{firstPlugin, secondPlugin} {
		if _, err := os.Stat(pluginPath(t, pluginDir, plugin)); err != nil {
			t.Errorf("expected %s from an active config root to survive clean: %v", plugin.Name, err)
		}
	}
}

func TestGatherPluginsExcludesParseOnlyDocumentAndItsNestedSources(t *testing.T) {
	pluginDir, parentConf := setupIntegrationDir(t)
	parseOnlyConf := filepath.Join(filepath.Dir(parentConf), "parse-only.conf")
	writeConf(t, parentConf, "set -g @plugin owner/active\nsource-file -n parse-only.conf\n")
	writeConf(t, parseOnlyConf, "set -g @plugin owner/ignored\nsource-file missing-nested.conf\n")

	paths := config.Paths{
		TmuxConf:   parentConf,
		PluginPath: mustRoot(t, pluginDir),
		Home:       filepath.Dir(parentConf),
	}
	plugins, err := config.GatherPlugins(&noopRunner{}, config.RealFS{}, paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 || plugins[0].Name != "owner/active" {
		t.Fatalf("plugins = %v, want only active parent declaration", plugins)
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
	p := mustParsePlugin(t, tmuxExamplePlugin)
	installedDir := pluginPath(t, pluginDir, p)
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
