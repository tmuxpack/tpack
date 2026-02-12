package config

import (
	"os"
	"path/filepath"

	"github.com/tmux-plugins/tpm/internal/tmux"
)

// Option is a functional option for Resolve.
type Option func(*resolveOpts)

type resolveOpts struct {
	fs   FS
	home string
	xdg  string // XDG_CONFIG_HOME override
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

func (o *resolveOpts) xdgConfigHome() string {
	if o.xdg != "" {
		return o.xdg
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(o.home, ".config")
}

// Resolve builds a Config by reading tmux options and checking filesystem paths.
// Priority for plugin path:
//  1. TMUX_PLUGIN_MANAGER_PATH env var already set in tmux
//  2. XDG config home (~/.config/tmux/tmux.conf exists → ~/.config/tmux/plugins/)
//  3. Default (~/.tmux/plugins/)
func Resolve(runner tmux.Runner, opts ...Option) (*Config, error) {
	o := &resolveOpts{
		fs:   RealFS{},
		home: os.Getenv("HOME"),
	}
	for _, opt := range opts {
		opt(o)
	}

	cfg := &Config{
		InstallKey: DefaultInstallKey,
		UpdateKey:  DefaultUpdateKey,
		CleanKey:   DefaultCleanKey,
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

	// Resolve tmux.conf location.
	cfg.TmuxConf = getUserTmuxConf(o)

	// Resolve plugin path.
	cfg.PluginPath = resolvePluginPath(runner, o)

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
