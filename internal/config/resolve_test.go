package config_test

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// testOpts returns common options that isolate tests from the real environment.
func testOpts(fs config.FS) []config.Option {
	return []config.Option{
		config.WithFS(fs),
		config.WithHome("/home/user"),
		config.WithXDG("/home/user/.config"),
	}
}

func TestResolveDefaults(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InstallKey != "I" {
		t.Errorf("InstallKey = %q, want %q", cfg.InstallKey, "I")
	}
	if cfg.UpdateKey != "U" {
		t.Errorf("UpdateKey = %q, want %q", cfg.UpdateKey, "U")
	}
	if cfg.CleanKey != "M-u" {
		t.Errorf("CleanKey = %q, want %q", cfg.CleanKey, "M-u")
	}
	if cfg.TmuxConf != "/home/user/.tmux.conf" {
		t.Errorf("TmuxConf = %q, want default", cfg.TmuxConf)
	}
	if cfg.PluginPath != "/home/user/.tmux/plugins/" {
		t.Errorf("PluginPath = %q, want default", cfg.PluginPath)
	}
}

func TestResolveCustomKeybindings(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm-install"] = "T"
	m.Options["@tpm-update"] = "Y"
	m.Options["@tpm-clean"] = "M-y"

	fs := config.NewMockFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InstallKey != "T" {
		t.Errorf("InstallKey = %q, want %q", cfg.InstallKey, "T")
	}
	if cfg.UpdateKey != "Y" {
		t.Errorf("UpdateKey = %q, want %q", cfg.UpdateKey, "Y")
	}
	if cfg.CleanKey != "M-y" {
		t.Errorf("CleanKey = %q, want %q", cfg.CleanKey, "M-y")
	}
}

func TestResolveXDGTmuxConf(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.config/tmux/tmux.conf"] = ""

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TmuxConf != "/home/user/.config/tmux/tmux.conf" {
		t.Errorf("TmuxConf = %q, want XDG path", cfg.TmuxConf)
	}
	if cfg.PluginPath != "/home/user/.config/tmux/plugins/" {
		t.Errorf("PluginPath = %q, want XDG plugins path", cfg.PluginPath)
	}
}

func TestResolvePluginPathFromEnv(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Environment["TMUX_PLUGIN_MANAGER_PATH"] = "/custom/path/"
	fs := config.NewMockFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PluginPath != "/custom/path/" {
		t.Errorf("PluginPath = %q, want %q", cfg.PluginPath, "/custom/path/")
	}
}

func TestResolvePluginPathTrailingSlash(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Environment["TMUX_PLUGIN_MANAGER_PATH"] = "/custom/path"
	fs := config.NewMockFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PluginPath != "/custom/path/" {
		t.Errorf("PluginPath = %q, want trailing slash", cfg.PluginPath)
	}
}
