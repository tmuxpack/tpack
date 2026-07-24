package config_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func testPaths(t *testing.T) config.Paths {
	return config.Paths{
		TmuxConf:      "/home/user/.tmux.conf",
		PluginPath:    mustRoot(t, "/home/user/.local/share/tmux/plugins"),
		Home:          "/home/user",
		XDGConfigHome: "/home/user/.config",
	}
}

func configWithPlugins(t *testing.T, content string) (config.FS, config.Paths) {
	t.Helper()
	paths := testPaths(t)
	fs := config.NewMockFS()
	fs.Files[paths.TmuxConf] = content
	return fs, paths
}

func TestGatherPluginsAllowsSameBasenameRepositories(t *testing.T) {
	fs, paths := configWithPlugins(t,
		`set -g @plugin "catppuccin/tmux"`+"\n"+
			`set -g @plugin "dracula/tmux"`)
	plugins, err := config.GatherPlugins(tmux.NewMockRunner(), fs, paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plugins[0].DirName == plugins[1].DirName {
		t.Fatalf("same-basename repositories share %q", plugins[0].DirName)
	}
}

func TestGatherPluginsRejectsConflictingAliases(t *testing.T) {
	fs, paths := configWithPlugins(t,
		`set -g @plugin "catppuccin/tmux alias=theme"`+"\n"+
			`set -g @plugin "dracula/tmux alias=theme"`)
	_, err := config.GatherPlugins(tmux.NewMockRunner(), fs, paths, nil)
	if err == nil || !strings.Contains(err.Error(), `directory "theme"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestGatherPluginsRejectsAliasWithPathSeparator(t *testing.T) {
	fs, paths := configWithPlugins(t,
		`set -g @plugin "catppuccin/tmux alias=group/theme"`+"\n"+
			`set -g @plugin "dracula/tmux alias=theme"`)
	_, err := config.GatherPlugins(tmux.NewMockRunner(), fs, paths, nil)
	if err == nil || !strings.Contains(err.Error(), `invalid plugin name "group/theme"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestGatherPluginsRejectsUnsafeAlias(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "owner/repo alias=.."`
	paths := testPaths(t)

	plugins, err := config.GatherPlugins(m, fs, paths, nil)
	if err == nil {
		t.Fatal("GatherPlugins returned no error")
	}
	if plugins != nil {
		t.Fatalf("plugins = %v, want nil", plugins)
	}
}

func mustRoot(t *testing.T, path string) plug.Root {
	t.Helper()
	root, err := plug.NewRoot("test", path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestGatherPluginsNewSyntax(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `
set -g @plugin "tmux-plugins/tpm"
set -g @plugin "tmux-plugins/tmux-sensible"
`

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name != "tmux-plugins/tpm" {
		t.Errorf("plugin[0].Name = %q, want %q", plugins[0].Name, "tmux-plugins/tpm")
	}
	if plugins[1].Name != "tmux-plugins/tmux-sensible" {
		t.Errorf("plugin[1].Name = %q, want %q", plugins[1].Name, "tmux-plugins/tmux-sensible")
	}
}

func TestGatherPluginsLegacySyntax(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm_plugins"] = "tmux-plugins/tpm tmux-plugins/tmux-yank"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = ""

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name != "tmux-plugins/tpm" {
		t.Errorf("plugin[0].Name = %q", plugins[0].Name)
	}
	if plugins[1].Name != "tmux-plugins/tmux-yank" {
		t.Errorf("plugin[1].Name = %q", plugins[1].Name)
	}
}

func TestGatherPluginsMixed(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm_plugins"] = "tmux-plugins/tpm"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "tmux-plugins/tmux-sensible"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestGatherPluginsFromSourcedFiles(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `
source ~/.tmux/plugins.conf
set -g @plugin "tmux-plugins/tpm"
`
	fs.Files["/home/user/.tmux/plugins.conf"] = `set -g @plugin "tmux-plugins/tmux-yank"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestGatherPluginsFromRecursiveSources(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "set -g @plugin owner/root\nsource ~/.tmux/one.conf"
	fs.Files["/home/user/.tmux/one.conf"] = "set -g @plugin owner/one\nsource ~/.tmux/two.conf"
	fs.Files["/home/user/.tmux/two.conf"] = "set -g @plugin owner/two"
	plugins, err := config.GatherPlugins(tmux.NewMockRunner(), fs, testPaths(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, plugin := range plugins {
		names = append(names, plugin.Name)
	}
	if want := []string{"owner/root", "owner/one", "owner/two"}; !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestGatherPluginsErrorsWhenRequiredSourceUnreadable(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source ~/.tmux/missing.conf"

	plugins, err := config.GatherPlugins(tmux.NewMockRunner(), fs, testPaths(t), nil)
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
	if plugins != nil {
		t.Fatalf("plugins = %v, want nil", plugins)
	}
}

func TestGatherPluginsAllowsMissingQuietSource(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -q ~/.tmux/missing.conf"

	plugins, err := config.GatherPlugins(tmux.NewMockRunner(), fs, testPaths(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Fatalf("plugins = %v, want none", plugins)
	}
}

func TestGatherPluginsIncludesEtcTmuxConf(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = `set -g @plugin "tmux-plugins/tmux-sensible"`
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "tmux-plugins/tpm"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestGatherPluginsEmpty(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = ""

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error for a readable-but-empty conf: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestGatherPluginsWithBranch(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "user/repo#develop"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Branch != "develop" {
		t.Errorf("Branch = %q, want %q", plugins[0].Branch, "develop")
	}
}

func TestGatherPluginsErrorsWhenTmuxConfUnreadable(t *testing.T) {
	// MockFS has no entry for the resolved TmuxConf: simulates a permission
	// error or a TOCTOU deletion between Resolve and GatherPlugins. This must
	// be an explicit, non-nil error -- never silently treated as "no plugins".
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	plugins, err := config.GatherPlugins(m, fs, testPaths(t), nil)
	if err == nil {
		t.Fatal("expected an error when the user tmux.conf cannot be read")
	}
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
	if plugins != nil {
		t.Errorf("expected nil plugins on error, got %v", plugins)
	}
}
