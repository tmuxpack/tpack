package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
)

func TestLegacyMigrationAllowsSameBasenameRepositories(t *testing.T) {
	skipIfNoGit(t)

	pluginDir, confFile := setupIntegrationDir(t)
	firstURL := "https://plugins.test/first/tmux.git"
	secondURL := "https://plugins.test/second/tmux.git"
	firstRepo := createLocalRepository(t, filepath.Join(t.TempDir(), "first", "tmux.git"), "first")
	secondRepo := createLocalRepository(t, filepath.Join(t.TempDir(), "second", "tmux.git"), "second")

	first := mustParsePlugin(t, firstURL)
	second := mustParsePlugin(t, secondURL)
	if plug.LegacyPluginName(first.Spec) != plug.LegacyPluginName(second.Spec) {
		t.Fatalf("test setup produced different legacy basenames: %q and %q", first.Spec, second.Spec)
	}
	if first.DirName == second.DirName {
		t.Fatalf("same-basename repositories produced the same directory %q", first.DirName)
	}

	legacyDir := legacyPluginPath(t, pluginDir, first)
	cloneRepository(t, firstRepo, legacyDir)
	setRepositoryOrigin(t, legacyDir, firstURL)
	writeConf(t, confFile, "set -g @plugin \""+firstURL+"\"\nset -g @plugin \""+secondURL+"\"\n")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	paths := config.Paths{
		TmuxConf:   confFile,
		PluginPath: mustRoot(t, pluginDir),
		Home:       t.TempDir(),
	}
	plugins, err := config.LoadPlugins(ctx, &noopRunner{}, config.RealFS{}, paths, gitcli.NewOriginReader(), nil)
	if err != nil {
		t.Fatalf("LoadPlugins() migration: %v", err)
	}

	firstDir := pluginPath(t, pluginDir, first)
	if _, err := os.Stat(firstDir); err != nil {
		t.Fatalf("migrated canonical directory: %v", err)
	}
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Fatalf("legacy directory still exists after migration: %v", err)
	}

	configureGitURLRewrites(t, map[string]string{
		firstURL:  firstRepo,
		secondURL: secondRepo,
	})
	output := ui.NewMockOutput()
	mgr := manager.New(paths.PluginPath, gitcli.NewCloner(), gitcli.NewPuller(), gitcli.NewValidator(), output)
	mgr.Install(ctx, plugins)
	if output.Result() != nil {
		t.Fatalf("Install() failed: %v", output.ErrMsgs)
	}

	assertMarker(t, firstDir, "first")
	assertMarker(t, pluginPath(t, pluginDir, second), "second")

	if _, err := config.LoadPlugins(ctx, &noopRunner{}, config.RealFS{}, paths, gitcli.NewOriginReader(), nil); err != nil {
		t.Fatalf("second LoadPlugins(): %v", err)
	}
	assertMarker(t, firstDir, "first")
	assertMarker(t, pluginPath(t, pluginDir, second), "second")
}
