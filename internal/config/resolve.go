package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/tmux-plugins/tpm/internal/tmux"
)

// Option is a functional option for Resolve.
type Option func(*resolveOpts)

type resolveOpts struct {
	fs       FS
	home     string
	xdg      string // XDG_CONFIG_HOME override
	xdgState string // XDG_STATE_HOME override
}

// WithFS overrides the filesystem for testing.
func WithFS(fs FS) Option {
	return func(o *resolveOpts) { o.fs = fs }
}

// WithHome overrides the home directory for testing.
func WithHome(home string) Option {
	return func(o *resolveOpts) { o.home = home }
}

// WithXDG overrides XDG_CONFIG_HOME for testing.
func WithXDG(xdg string) Option {
	return func(o *resolveOpts) { o.xdg = xdg }
}

// WithXDGState overrides XDG_STATE_HOME for testing.
func WithXDGState(xdgState string) Option {
	return func(o *resolveOpts) { o.xdgState = xdgState }
}

func (o *resolveOpts) xdgConfigHome() string {
	if o.xdg != "" {
		return o.xdg
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(o.home, ".config")
}

func (o *resolveOpts) xdgStateHome() string {
	if o.xdgState != "" {
		return o.xdgState
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	return filepath.Join(o.home, ".local", "state")
}

// Resolve builds a Config by reading tmux options and checking filesystem paths.
// Priority for plugin path:
//  1. TMUX_PLUGIN_MANAGER_PATH env var already set in tmux
//  2. XDG config home (~/.config/tmux/tmux.conf exists → ~/.config/tmux/plugins/)
//  3. Default (~/.tmux/plugins/)
func Resolve(runner tmux.Runner, opts ...Option) (*Config, error) {
	home := os.Getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	o := &resolveOpts{
		fs:   RealFS{},
		home: home,
	}
	for _, opt := range opts {
		opt(o)
	}

	cfg := &Config{
		InstallKey: DefaultInstallKey,
		UpdateKey:  DefaultUpdateKey,
		CleanKey:   DefaultCleanKey,
		TuiKey:     DefaultTuiKey,
	}

	// Resolve keybindings from tmux options.
	if v, err := runner.ShowOption(InstallKeyOption); err == nil && v != "" {
		cfg.InstallKey = v
	}
	if v, err := runner.ShowOption(UpdateKeyOption); err == nil && v != "" {
		cfg.UpdateKey = v
	}
	if v, err := runner.ShowOption(CleanKeyOption); err == nil && v != "" {
		cfg.CleanKey = v
	}
	if v, err := runner.ShowOption(TuiKeyOption); err == nil && v != "" {
		cfg.TuiKey = v
	}

	// Resolve tmux.conf location.
	cfg.TmuxConf = getUserTmuxConf(o)

	// Resolve plugin path.
	cfg.PluginPath = resolvePluginPath(runner, o)

	// Resolve color and update overrides from tmux options.
	cfg.Colors = resolveColors(runner)
	cfg.UpdateCheckInterval, cfg.UpdateMode = resolveUpdateSettings(runner)

	if v, err := runner.ShowOption(VersionOption); err == nil && v != "" {
		cfg.PinnedVersion = v
	}

	cfg.StatePath = filepath.Join(o.xdgStateHome(), "tpm")

	cfg.Home = o.home

	return cfg, nil
}

// getUserTmuxConf returns the user's tmux.conf path (XDG first, then default).
func getUserTmuxConf(o *resolveOpts) string {
	xdgConf := filepath.Join(o.xdgConfigHome(), "tmux", "tmux.conf")
	if o.fs.FileExists(xdgConf) {
		return xdgConf
	}
	return filepath.Join(o.home, ".tmux.conf")
}

// resolvePluginPath determines the plugin installation directory.
func resolvePluginPath(runner tmux.Runner, o *resolveOpts) string {
	// Check if already set in tmux environment.
	if val, err := runner.ShowEnvironment(TPMEnvVar); err == nil && val != "" && val != "/" {
		if val[len(val)-1] != '/' {
			val += "/"
		}
		return val
	}

	// Check XDG path.
	xdgConf := filepath.Join(o.xdgConfigHome(), "tmux", "tmux.conf")
	if o.fs.FileExists(xdgConf) {
		return filepath.Join(o.xdgConfigHome(), "tmux", "plugins") + "/"
	}

	// Default.
	return filepath.Join(o.home, DefaultTPMPath) + "/"
}

// resolveColors reads per-color tmux options into a ColorConfig.
func resolveColors(runner tmux.Runner) ColorConfig {
	var c ColorConfig
	if v, err := runner.ShowOption(ColorPrimaryOption); err == nil && v != "" {
		c.Primary = v
	}
	if v, err := runner.ShowOption(ColorSecondaryOption); err == nil && v != "" {
		c.Secondary = v
	}
	if v, err := runner.ShowOption(ColorAccentOption); err == nil && v != "" {
		c.Accent = v
	}
	if v, err := runner.ShowOption(ColorErrorOption); err == nil && v != "" {
		c.Error = v
	}
	if v, err := runner.ShowOption(ColorMutedOption); err == nil && v != "" {
		c.Muted = v
	}
	if v, err := runner.ShowOption(ColorTextOption); err == nil && v != "" {
		c.Text = v
	}
	return c
}

// resolveUpdateSettings reads update interval and mode from tmux options.
func resolveUpdateSettings(runner tmux.Runner) (time.Duration, string) {
	var interval time.Duration
	var mode string
	if v, err := runner.ShowOption(UpdateIntervalOption); err == nil && v != "" {
		interval = parseCheckInterval(v)
	}
	if v, err := runner.ShowOption(UpdateModeOption); err == nil && v != "" {
		mode = parseUpdateMode(v)
	}
	return interval, mode
}

// validUpdateModes is the set of recognized update mode values.
var validUpdateModes = map[string]bool{
	"":       true,
	"off":    true,
	"prompt": true,
	"auto":   true,
}

// parseUpdateMode returns the mode if valid, or empty string otherwise.
func parseUpdateMode(s string) string {
	if validUpdateModes[s] {
		return s
	}
	return ""
}

// parseCheckInterval parses a duration string, returning 0 on any error.
func parseCheckInterval(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0
	}
	return d
}
