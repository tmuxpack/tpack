package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestGatherPluginsDoesNotMigrate(t *testing.T) {
	paths, legacyPath := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux"`)

	plugins, err := config.GatherPlugins(tmux.NewMockRunner(), config.RealFS{}, paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("plugins = %v, want one", plugins)
	}
	assertPathExists(t, legacyPath)
}

func TestLoadPluginsMigratesAfterValidGather(t *testing.T) {
	paths, legacyPath := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux"`)
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	plugins, err := config.LoadPlugins(context.Background(), tmux.NewMockRunner(), config.RealFS{}, paths, origins, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("plugins = %v, want one", plugins)
	}
	assertPathMissing(t, legacyPath)
	canonicalPath, err := paths.PluginPath.Child(plugins[0].DirName)
	if err != nil {
		t.Fatal(err)
	}
	assertPathExists(t, canonicalPath)
}

func TestLoadPluginsInvalidConfigDoesNotInspectOrigins(t *testing.T) {
	paths, _ := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux alias=.."`)
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	plugins, err := config.LoadPlugins(context.Background(), tmux.NewMockRunner(), config.RealFS{}, paths, origins, nil)
	if err == nil {
		t.Fatal("expected config error")
	}
	if plugins != nil {
		t.Fatalf("plugins = %v, want nil", plugins)
	}
	if len(origins.Calls) != 0 {
		t.Fatalf("origin calls = %v, want none", origins.Calls)
	}
}

func migrationTestPaths(t *testing.T, content string) (config.Paths, string) {
	t.Helper()
	base := t.TempDir()
	conf := filepath.Join(base, "tmux.conf")
	if err := os.WriteFile(conf, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(base, "plugins")
	legacyPath := filepath.Join(rootPath, plug.LegacyPluginName("catppuccin/tmux"))
	if err := os.MkdirAll(legacyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	return config.Paths{
		TmuxConf:      conf,
		PluginPath:    mustRoot(t, rootPath),
		Home:          base,
		XDGConfigHome: filepath.Join(base, ".config"),
	}, legacyPath
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("path %q does not exist: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path %q still exists: %v", path, err)
	}
}
