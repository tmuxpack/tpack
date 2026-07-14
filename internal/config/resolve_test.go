package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// testOpts returns common options that isolate tests from the real environment.
func testOpts(fs config.FS) []config.Option {
	return []config.Option{
		config.WithFS(fs),
		config.WithEnv(config.Env{Home: "/home/user", XDGConfigHome: "/home/user/.config"}),
	}
}

// newTestFS returns a MockFS with a tmux.conf seeded so Resolve succeeds.
func newTestFS() *config.MockFS {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = ""
	return fs
}

func TestResolveEmptyHomeReturnsError(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()

	_, err := config.Resolve(m, config.WithFS(fs), config.WithEnv(config.Env{}))
	if err == nil {
		t.Fatal("expected error for empty home, got nil")
	}
}

func TestResolveDefaults(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := newTestFS()

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
	if cfg.PluginPath != "/home/user/.local/share/tmux/plugins/" {
		t.Errorf("PluginPath = %q, want XDG data default", cfg.PluginPath)
	}
	if cfg.Paths.PluginPathSource != config.SourceDefaultXDGData {
		t.Errorf("PluginPathSource = %d, want SourceDefaultXDGData", cfg.Paths.PluginPathSource)
	}
	if cfg.PinnedVersion != "" {
		t.Errorf("PinnedVersion = %q, want empty", cfg.PinnedVersion)
	}
}

func TestResolveCustomKeybindings(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpack-install"] = "T"
	m.Options["@tpack-update"] = "Y"
	m.Options["@tpack-clean"] = "M-y"
	m.Options["@tpack-tui"] = "P"

	fs := newTestFS()

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

func TestResolveLegacyKeybindings(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpm-install"] = "X"
	m.Options["@tpm-update"] = "Z"
	m.Options["@tpm-clean"] = "M-z"

	fs := newTestFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InstallKey != "X" {
		t.Errorf("InstallKey = %q, want %q", cfg.InstallKey, "X")
	}
	if cfg.UpdateKey != "Z" {
		t.Errorf("UpdateKey = %q, want %q", cfg.UpdateKey, "Z")
	}
	if cfg.CleanKey != "M-z" {
		t.Errorf("CleanKey = %q, want %q", cfg.CleanKey, "M-z")
	}
}

func TestResolveCurrentKeybindingsOverrideLegacy(t *testing.T) {
	m := tmux.NewMockRunner()
	// Set both legacy and current; current should win.
	m.Options["@tpm-install"] = "X"
	m.Options["@tpack-install"] = "T"
	m.Options["@tpm-update"] = "Z"
	m.Options["@tpack-update"] = "Y"
	m.Options["@tpm-clean"] = "M-z"
	m.Options["@tpack-clean"] = "M-y"

	fs := newTestFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InstallKey != "T" {
		t.Errorf("InstallKey = %q, want %q (current should override legacy)", cfg.InstallKey, "T")
	}
	if cfg.UpdateKey != "Y" {
		t.Errorf("UpdateKey = %q, want %q (current should override legacy)", cfg.UpdateKey, "Y")
	}
	if cfg.CleanKey != "M-y" {
		t.Errorf("CleanKey = %q, want %q (current should override legacy)", cfg.CleanKey, "M-y")
	}
}

func TestResolveXDGTmuxConf(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	fs.Files["/home/user/.config/tmux/tmux.conf"] = ""
	fs.Files["/home/user/.config/tmux/plugins"] = ""

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
	fs := newTestFS()

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
	fs := newTestFS()

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
	fs := newTestFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/home/user/.local/state/tpack"
	if cfg.StatePath != want {
		t.Errorf("StatePath = %q, want %q", cfg.StatePath, want)
	}
}

func TestResolveStatePathWithXDGState(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := newTestFS()

	cfg, err := config.Resolve(m,
		config.WithFS(fs),
		config.WithEnv(config.Env{
			Home:          "/home/user",
			XDGConfigHome: "/home/user/.config",
			XDGStateHome:  "/custom/state",
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/custom/state/tpack"
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
				"@tpack-color-primary":   "#111111",
				"@tpack-color-secondary": "#222222",
				"@tpack-color-accent":    "#333333",
				"@tpack-color-error":     "#444444",
				"@tpack-color-muted":     "#555555",
				"@tpack-color-text":      "#666666",
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
				"@tpack-color-primary": "#aabbcc",
				"@tpack-color-text":    "#ddeeff",
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
			fs := newTestFS()

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
				"@tpack-update-interval": "24h",
				"@tpack-update-mode":     "prompt",
			},
			wantInterval: 24 * time.Hour,
			wantMode:     "prompt",
		},
		{
			name: "auto mode with 1h interval",
			options: map[string]string{
				"@tpack-update-interval": "1h",
				"@tpack-update-mode":     "auto",
			},
			wantInterval: 1 * time.Hour,
			wantMode:     "auto",
		},
		{
			name: "off mode",
			options: map[string]string{
				"@tpack-update-mode": "off",
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
				"@tpack-update-interval": "not-a-duration",
				"@tpack-update-mode":     "prompt",
			},
			wantInterval: 0,
			wantMode:     "prompt",
		},
		{
			name: "invalid mode ignored",
			options: map[string]string{
				"@tpack-update-mode": "bogus",
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
			fs := newTestFS()

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
				"@tpack-version": "v1.2.3",
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
			fs := newTestFS()

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
	fs := newTestFS()

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
	if cfg.PinnedVersion != "" {
		t.Errorf("PinnedVersion = %q, want empty", cfg.PinnedVersion)
	}
	if len(cfg.HiddenCategories) != 0 {
		t.Errorf("HiddenCategories = %v, want nil", cfg.HiddenCategories)
	}
}

func TestResolveHiddenCategories(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"single category", "ai", []string{"ai"}},
		{"multiple categories", "ai,development", []string{"ai", "development"}},
		{"with spaces", " ai , development ", []string{"ai", "development"}},
		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			if tt.value != "" {
				m.Options["@tpack-hidden-categories"] = tt.value
			}
			fs := newTestFS()

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(cfg.HiddenCategories) != len(tt.want) {
				t.Fatalf("HiddenCategories = %v, want %v", cfg.HiddenCategories, tt.want)
			}
			for i, got := range cfg.HiddenCategories {
				if got != tt.want[i] {
					t.Errorf("HiddenCategories[%d] = %q, want %q", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestResolveFailsClosedWithoutTmuxConf(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS() // deliberately empty: no tmux.conf anywhere

	_, err := config.Resolve(m, testOpts(fs)...)

	var noConf *config.ErrNoTmuxConf
	if !errors.As(err, &noConf) {
		t.Fatalf("expected ErrNoTmuxConf, got %v", err)
	}
}

func TestResolvePluginPathOption(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@tpack-plugin-path"] = "/opt/plugins"
	m.Environment["TMUX_PLUGIN_MANAGER_PATH"] = "/env/legacy/"
	fs := newTestFS()

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PluginPath != "/opt/plugins/" {
		t.Errorf("PluginPath = %q, want option value to win", cfg.PluginPath)
	}
	if cfg.Paths.PluginPathSource != config.SourceOption {
		t.Errorf("PluginPathSource = %d, want SourceOption", cfg.Paths.PluginPathSource)
	}
}
