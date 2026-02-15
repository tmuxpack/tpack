package config_test

import (
	"testing"
	"time"

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
	if cfg.TuiKey != "T" {
		t.Errorf("TuiKey = %q, want %q", cfg.TuiKey, "T")
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
	m.Options["@tpm-tui"] = "P"

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
	if cfg.TuiKey != "P" {
		t.Errorf("TuiKey = %q, want %q", cfg.TuiKey, "P")
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

func TestResolveStatePath(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	t.Setenv("XDG_STATE_HOME", "")

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/home/user/.local/state/tpm"
	if cfg.StatePath != want {
		t.Errorf("StatePath = %q, want %q", cfg.StatePath, want)
	}
}

func TestResolveStatePathWithXDGState(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	t.Setenv("XDG_STATE_HOME", "/custom/state")

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/custom/state/tpm"
	if cfg.StatePath != want {
		t.Errorf("StatePath = %q, want %q", cfg.StatePath, want)
	}
}

func TestResolveColors(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    config.ColorConfig
	}{
		{
			name: "all colors set",
			options: map[string]string{
				"@tpm-color-primary":   "#111111",
				"@tpm-color-secondary": "#222222",
				"@tpm-color-accent":    "#333333",
				"@tpm-color-error":     "#444444",
				"@tpm-color-muted":     "#555555",
				"@tpm-color-text":      "#666666",
			},
			want: config.ColorConfig{
				Primary:   "#111111",
				Secondary: "#222222",
				Accent:    "#333333",
				Error:     "#444444",
				Muted:     "#555555",
				Text:      "#666666",
			},
		},
		{
			name: "partial colors",
			options: map[string]string{
				"@tpm-color-primary": "#aabbcc",
				"@tpm-color-text":    "#ddeeff",
			},
			want: config.ColorConfig{
				Primary: "#aabbcc",
				Text:    "#ddeeff",
			},
		},
		{
			name:    "no colors set",
			options: map[string]string{},
			want:    config.ColorConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			for k, v := range tt.options {
				m.Options[k] = v
			}
			fs := config.NewMockFS()

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Colors != tt.want {
				t.Errorf("Colors = %+v, want %+v", cfg.Colors, tt.want)
			}
		})
	}
}

func TestResolveUpdateSettings(t *testing.T) {
	tests := []struct {
		name         string
		options      map[string]string
		wantInterval time.Duration
		wantMode     string
	}{
		{
			name: "prompt mode with 24h interval",
			options: map[string]string{
				"@tpm-update-interval": "24h",
				"@tpm-update-mode":     "prompt",
			},
			wantInterval: 24 * time.Hour,
			wantMode:     "prompt",
		},
		{
			name: "auto mode with 1h interval",
			options: map[string]string{
				"@tpm-update-interval": "1h",
				"@tpm-update-mode":     "auto",
			},
			wantInterval: 1 * time.Hour,
			wantMode:     "auto",
		},
		{
			name: "off mode",
			options: map[string]string{
				"@tpm-update-mode": "off",
			},
			wantInterval: 0,
			wantMode:     "off",
		},
		{
			name:         "no update options",
			options:      map[string]string{},
			wantInterval: 0,
			wantMode:     "",
		},
		{
			name: "invalid interval",
			options: map[string]string{
				"@tpm-update-interval": "not-a-duration",
				"@tpm-update-mode":     "prompt",
			},
			wantInterval: 0,
			wantMode:     "prompt",
		},
		{
			name: "invalid mode ignored",
			options: map[string]string{
				"@tpm-update-mode": "bogus",
			},
			wantInterval: 0,
			wantMode:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			for k, v := range tt.options {
				m.Options[k] = v
			}
			fs := config.NewMockFS()

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.UpdateCheckInterval != tt.wantInterval {
				t.Errorf("UpdateCheckInterval = %v, want %v", cfg.UpdateCheckInterval, tt.wantInterval)
			}
			if cfg.UpdateMode != tt.wantMode {
				t.Errorf("UpdateMode = %q, want %q", cfg.UpdateMode, tt.wantMode)
			}
		})
	}
}

func TestResolvePinnedVersion(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    string
	}{
		{
			name:    "no pinned version",
			options: map[string]string{},
			want:    "",
		},
		{
			name: "pinned to specific version",
			options: map[string]string{
				"@tpm-version": "v1.2.3",
			},
			want: "v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			for k, v := range tt.options {
				m.Options[k] = v
			}
			fs := config.NewMockFS()

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.PinnedVersion != tt.want {
				t.Errorf("PinnedVersion = %q, want %q", cfg.PinnedVersion, tt.want)
			}
		})
	}
}

func TestResolveDefaultsNoColors(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Colors != (config.ColorConfig{}) {
		t.Errorf("Colors = %+v, want zero value", cfg.Colors)
	}
	if cfg.UpdateCheckInterval != 0 {
		t.Errorf("UpdateCheckInterval = %v, want 0", cfg.UpdateCheckInterval)
	}
	if cfg.UpdateMode != "" {
		t.Errorf("UpdateMode = %q, want empty", cfg.UpdateMode)
	}
}
