package config

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/tmuxpack/tpack/internal/tmux"
)

// Option is a functional option for Resolve.
type Option func(*resolveOpts)

type resolveOpts struct {
	fs     FS
	env    Env
	envSet bool
}

// WithFS overrides the filesystem
func WithFS(fs FS) Option {
	return func(o *resolveOpts) { o.fs = fs }
}

// WithEnv overrides the process environment used for path resolution.
func WithEnv(env Env) Option {
	return func(o *resolveOpts) { o.env = env; o.envSet = true }
}

// Resolve builds a Config by reading tmux options and resolving paths.
// Path precedence and tmux.conf discovery are documented on ResolvePaths.
func Resolve(runner tmux.Runner, opts ...Option) (*Config, error) {
	o := &resolveOpts{fs: RealFS{}}
	for _, opt := range opts {
		opt(o)
	}
	if !o.envSet {
		o.env = EnvFromOS()
	}

	paths, err := ResolvePaths(runner, o.fs, o.env)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		InstallKey: DefaultInstallKey,
		UpdateKey:  DefaultUpdateKey,
		CleanKey:   DefaultCleanKey,
		TuiKey:     DefaultTuiKey,
	}

	// current @tpack-* first, legacy @tpm-* fallback
	cfg.InstallKey = resolveOptionWithLegacyAndFallback(runner, InstallKeyOption, LegacyInstallKeyOption, cfg.InstallKey)
	cfg.UpdateKey = resolveOptionWithLegacyAndFallback(runner, UpdateKeyOption, LegacyUpdateKeyOption, cfg.UpdateKey)
	cfg.CleanKey = resolveOptionWithLegacyAndFallback(runner, CleanKeyOption, LegacyCleanKeyOption, cfg.CleanKey)
	cfg.TuiKey = resolveOptionWithFallback(runner, TuiKeyOption, cfg.TuiKey)

	cfg.Paths = paths
	cfg.TmuxConf = paths.TmuxConf
	cfg.PluginPath = paths.PluginPath
	cfg.Home = paths.Home
	cfg.Colors = resolveColors(runner)
	cfg.UpdateCheckInterval, cfg.UpdateMode = resolveUpdateSettings(runner)

	if v, err := runner.ShowOption(VersionOption); err == nil && v != "" {
		cfg.PinnedVersion = v
	}

	cfg.HiddenCategories = resolveHiddenCategories(runner)
	cfg.StatePath = filepath.Join(o.env.stateHome(), "tpack")

	return cfg, nil
}

// Reads per-color tmux options into a ColorConfig.
func resolveColors(runner tmux.Runner) ColorConfig {
	var c ColorConfig

	for _, entry := range []struct {
		option string
		field  *string
	}{
		{ColorPrimaryOption, &c.Primary},
		{ColorSecondaryOption, &c.Secondary},
		{ColorAccentOption, &c.Accent},
		{ColorErrorOption, &c.Error},
		{ColorMutedOption, &c.Muted},
		{ColorTextOption, &c.Text},
	} {
		if v, err := runner.ShowOption(entry.option); err == nil && v != "" {
			*entry.field = v
		}
	}

	return c
}

// Reads update interval and mode from tmux options.
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

var validUpdateModes = map[string]bool{
	"":       true,
	"off":    true,
	"prompt": true,
	"auto":   true,
}

// Returns the mode if valid, or empty string otherwise.
func parseUpdateMode(s string) string {
	if validUpdateModes[s] {
		return s
	}
	return ""
}

// Parses a duration string, returning 0 on any error.
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

// Reads a tmux option, falling back to a legacy name.
// Returns the default if neither is set.
func resolveOptionWithLegacyAndFallback(runner tmux.Runner, current, legacy, def string) string {
	if v, err := runner.ShowOption(current); err == nil && v != "" {
		return v
	}

	if legacy != "" {
		if v, err := runner.ShowOption(legacy); err == nil && v != "" {
			return v
		}
	}

	return def
}

func resolveOptionWithFallback(runner tmux.Runner, current, def string) string {
	if v, err := runner.ShowOption(current); err == nil && v != "" {
		return v
	}

	return def
}

// resolveHiddenCategories reads a comma-separated list of category names to hide.
func resolveHiddenCategories(runner tmux.Runner) []string {
	v, err := runner.ShowOption(HiddenCategoriesOption)
	if err != nil || v == "" {
		return nil
	}
	var cats []string
	for _, c := range strings.Split(v, ",") {
		c = strings.TrimSpace(c)
		if c != "" {
			cats = append(cats, c)
		}
	}
	return cats
}
