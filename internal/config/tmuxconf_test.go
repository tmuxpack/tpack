package config_test

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func testPaths() config.Paths {
	return config.Paths{
		TmuxConf:      "/home/user/.tmux.conf",
		Home:          "/home/user",
		XDGConfigHome: "/home/user/.config",
	}
}

func TestGatherPluginsNewSyntax(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `
set -g @plugin "tmux-plugins/tpm"
set -g @plugin "tmux-plugins/tmux-sensible"
`

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name != "tpm" {
		t.Errorf("plugin[0].Name = %q, want %q", plugins[0].Name, "tpm")
	}
	if plugins[1].Name != "tmux-sensible" {
		t.Errorf("plugin[1].Name = %q, want %q", plugins[1].Name, "tmux-sensible")
	}
}

func TestGatherPluginsLegacySyntax(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm_plugins"] = "tmux-plugins/tpm tmux-plugins/tmux-yank"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = ""

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name != "tpm" {
		t.Errorf("plugin[0].Name = %q", plugins[0].Name)
	}
	if plugins[1].Name != "tmux-yank" {
		t.Errorf("plugin[1].Name = %q", plugins[1].Name)
	}
}

func TestGatherPluginsMixed(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm_plugins"] = "tmux-plugins/tpm"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "tmux-plugins/tmux-sensible"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
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

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestGatherPluginsIncludesEtcTmuxConf(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = `set -g @plugin "tmux-plugins/tmux-sensible"`
	fs.Files["/home/user/.tmux.conf"] = `set -g @plugin "tmux-plugins/tpm"`

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
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

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
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

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
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

	plugins, err := config.GatherPlugins(m, fs, testPaths(), nil)
	if err == nil {
		t.Fatal("expected an error when the user tmux.conf cannot be read")
	}
	if plugins != nil {
		t.Errorf("expected nil plugins on error, got %v", plugins)
	}
}
