package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// PathSource identifies where a resolved path came from.
type PathSource int

const (
	// SourceOption is the @tpack-plugin-path tmux option.
	SourceOption PathSource = iota
	// SourceEnvTpack is the TPACK_PLUGIN_PATH tmux environment variable.
	SourceEnvTpack
	// SourceEnvLegacy is the TMUX_PLUGIN_MANAGER_PATH tmux environment variable.
	SourceEnvLegacy
	// SourceDetectedXDGConfig is an existing $XDG_CONFIG_HOME/tmux/plugins/ directory.
	SourceDetectedXDGConfig
	// SourceDetectedLegacy is an existing ~/.tmux/plugins/ directory.
	SourceDetectedLegacy
	// SourceDefaultXDGData is the $XDG_DATA_HOME/tmux/plugins/ default.
	SourceDefaultXDGData
)

// Env carries process-environment inputs for path resolution.
// Empty XDG fields fall back to the spec defaults derived from Home.
type Env struct {
	Home          string
	XDGConfigHome string
	XDGDataHome   string
	XDGStateHome  string
}

// EnvFromOS reads the environment variables used for path resolution.
func EnvFromOS() Env {
	home := os.Getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	return Env{
		Home:          home,
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:  os.Getenv("XDG_STATE_HOME"),
	}
}

func (e Env) configHome() string {
	if e.XDGConfigHome != "" {
		return e.XDGConfigHome
	}
	return filepath.Join(e.Home, ".config")
}

func (e Env) dataHome() string {
	if e.XDGDataHome != "" {
		return e.XDGDataHome
	}
	return filepath.Join(e.Home, ".local", "share")
}

func (e Env) stateHome() string {
	if e.XDGStateHome != "" {
		return e.XDGStateHome
	}
	return filepath.Join(e.Home, ".local", "state")
}

// ErrNoTmuxConf reports that no tmux.conf exists at any searched location.
type ErrNoTmuxConf struct {
	Searched []string
}

func (e *ErrNoTmuxConf) Error() string {
	return "no tmux.conf found (searched: " + strings.Join(e.Searched, ", ") + ")"
}

// Paths holds every resolved path with its provenance.
type Paths struct {
	// TmuxConf is the user's tmux.conf; guaranteed to exist on disk.
	TmuxConf string
	// ConfSearched lists the candidates tried, deduplicated, for error messages.
	ConfSearched []string
	// PluginPath is the plugin directory, trailing-slash normalised.
	PluginPath string
	// PluginPathSource records which precedence tier produced PluginPath.
	PluginPathSource PathSource

	Home          string
	XDGConfigHome string
	XDGDataHome   string
}

// ResolvePaths locates the user's tmux.conf and the plugin directory.
//
// tmux.conf discovery mirrors tmux 3.2+'s search order; a missing config is
// an error, never an empty result. Plugin path precedence:
//
//  1. @tpack-plugin-path tmux option
//  2. TPACK_PLUGIN_PATH tmux environment
//  3. TMUX_PLUGIN_MANAGER_PATH tmux environment
//  4. existing plugin directory, preferring the one adjacent to the resolved tmux.conf
//  5. $XDG_DATA_HOME/tmux/plugins/
func ResolvePaths(runner tmux.Runner, fs FS, env Env) (Paths, error) {
	if env.Home == "" {
		return Paths{}, errors.New("could not determine home directory")
	}

	p := Paths{
		Home:          env.Home,
		XDGConfigHome: env.configHome(),
		XDGDataHome:   env.dataHome(),
	}

	conf, searched := findTmuxConf(fs, p)
	if conf == "" {
		return Paths{ConfSearched: searched}, &ErrNoTmuxConf{Searched: searched}
	}
	p.TmuxConf = conf
	p.ConfSearched = searched

	p.PluginPath, p.PluginPathSource = resolvePluginDir(runner, fs, p)
	return p, nil
}

// findTmuxConf returns the first existing candidate, mirroring tmux's order.
func findTmuxConf(fs FS, p Paths) (string, []string) {
	candidates := []string{
		filepath.Join(p.XDGConfigHome, "tmux", "tmux.conf"),
		filepath.Join(p.Home, ".config", "tmux", "tmux.conf"),
		filepath.Join(p.Home, ".tmux.conf"),
	}

	seen := make(map[string]bool, len(candidates))
	var searched []string
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		seen[c] = true
		searched = append(searched, c)
	}

	for _, c := range searched {
		if fs.FileExists(c) {
			return c, searched
		}
	}
	return "", searched
}

func resolvePluginDir(runner tmux.Runner, fs FS, p Paths) (string, PathSource) {
	if v, err := runner.ShowOption(PluginPathOption); err == nil && validPluginPath(v) {
		return withTrailingSlash(plug.ManualExpansion(v, p.Home, p.XDGConfigHome)), SourceOption
	}
	if v, err := runner.ShowEnvironment(PluginPathEnvVar); err == nil && validPluginPath(v) {
		return withTrailingSlash(v), SourceEnvTpack
	}
	if v, err := runner.ShowEnvironment(LegacyPluginPathEnvVar); err == nil && validPluginPath(v) {
		return withTrailingSlash(v), SourceEnvLegacy
	}

	ordered := []struct {
		dir string
		src PathSource
	}{
		{filepath.Join(p.XDGConfigHome, "tmux", "plugins"), SourceDetectedXDGConfig},
		{filepath.Join(p.Home, ".tmux", "plugins"), SourceDetectedLegacy},
	}
	// Prefer the directory adjacent to the resolved tmux.conf.
	if p.TmuxConf == filepath.Join(p.Home, ".tmux.conf") {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	for _, cand := range ordered {
		if fs.FileExists(cand.dir) {
			return withTrailingSlash(cand.dir), cand.src
		}
	}

	return withTrailingSlash(filepath.Join(p.XDGDataHome, "tmux", "plugins")), SourceDefaultXDGData
}

func validPluginPath(v string) bool {
	return v != "" && v != "/"
}

func withTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}
