package config_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

const wantMigrationWarning = "Plugin paths changed during migration; restart tmux and update scripts that use old plugin paths if needed."

type originReaderFunc func(context.Context, string) (string, error)

func (f originReaderFunc) Origin(ctx context.Context, dir string) (string, error) {
	return f(ctx, dir)
}

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

func TestLoadPluginsWarnsOnceAfterMultipleRenames(t *testing.T) {
	paths, _ := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux"
set -g @plugin "tmux-plugins/tmux-sensible"`)
	secondLegacy, err := paths.PluginPath.Child(plug.LegacyPluginName("tmux-plugins/tmux-sensible"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(secondLegacy, 0o700); err != nil {
		t.Fatal(err)
	}
	origins := originReaderFunc(func(_ context.Context, dir string) (string, error) {
		urls := map[string]string{
			"tmux":          "git@github.com:catppuccin/tmux.git",
			"tmux-sensible": "git@github.com:tmux-plugins/tmux-sensible.git",
		}
		return urls[filepath.Base(dir)], nil
	})
	var warnings []string

	plugins, err := config.LoadPlugins(context.Background(), tmux.NewMockRunner(), config.RealFS{}, paths, origins,
		func(message string) { warnings = append(warnings, message) })
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 2 {
		t.Fatalf("plugins = %v, want two", plugins)
	}
	if len(warnings) != 1 || warnings[0] != wantMigrationWarning {
		t.Fatalf("warnings = %q, want [%q]", warnings, wantMigrationWarning)
	}
}

func TestLoadPluginsDoesNotWarnWithoutRename(t *testing.T) {
	paths, legacyPath := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux"`)
	if err := os.RemoveAll(legacyPath); err != nil {
		t.Fatal(err)
	}
	var warnings []string

	_, err := config.LoadPlugins(context.Background(), tmux.NewMockRunner(), config.RealFS{}, paths,
		&git.MockOriginReader{}, func(message string) { warnings = append(warnings, message) })
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
}

func TestLoadPluginsWarnsWhenEarlierRenamePrecedesFailure(t *testing.T) {
	paths, _ := migrationTestPaths(t, `set -g @plugin "catppuccin/tmux"
set -g @plugin "tmux-plugins/tmux-sensible"`)
	secondLegacy, err := paths.PluginPath.Child(plug.LegacyPluginName("tmux-plugins/tmux-sensible"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(secondLegacy, 0o700); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("origin failed")
	origins := originReaderFunc(func(_ context.Context, dir string) (string, error) {
		if filepath.Base(dir) == "tmux-sensible" {
			return "", wantErr
		}
		return "git@github.com:catppuccin/tmux.git", nil
	})
	var warnings []string

	plugins, err := config.LoadPlugins(context.Background(), tmux.NewMockRunner(), config.RealFS{}, paths, origins,
		func(message string) { warnings = append(warnings, message) })
	if plugins != nil {
		t.Fatalf("plugins = %v, want nil", plugins)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
	if len(warnings) != 1 || warnings[0] != wantMigrationWarning {
		t.Fatalf("warnings = %q, want [%q]", warnings, wantMigrationWarning)
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
