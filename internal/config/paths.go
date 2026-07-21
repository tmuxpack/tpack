package config

import (
	"fmt"
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
	// PluginPath is the validated plugin directory.
	PluginPath plug.Root
	// PluginPathSource records which precedence tier produced PluginPath.
	PluginPathSource PathSource

	Home          string
	XDGConfigHome string
	XDGDataHome   string
	XDGStateHome  string
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
	home, err := absoluteDir("HOME", env.Home)
	if err != nil {
		return Paths{}, err
	}
	configHome, err := resolveHome("XDG_CONFIG_HOME", env.XDGConfigHome, filepath.Join(home, ".config"))
	if err != nil {
		return Paths{}, err
	}
	dataHome, err := resolveHome("XDG_DATA_HOME", env.XDGDataHome, filepath.Join(home, ".local", "share"))
	if err != nil {
		return Paths{}, err
	}
	stateHome, err := resolveHome("XDG_STATE_HOME", env.XDGStateHome, filepath.Join(home, ".local", "state"))
	if err != nil {
		return Paths{}, err
	}

	p := Paths{
		Home:          home,
		XDGConfigHome: configHome,
		XDGDataHome:   dataHome,
		XDGStateHome:  stateHome,
	}

	conf, searched := findTmuxConf(fs, p)
	if conf == "" {
		return Paths{ConfSearched: searched}, &ErrNoTmuxConf{Searched: searched}
	}
	p.TmuxConf = conf
	p.ConfSearched = searched

	p.PluginPath, p.PluginPathSource, err = resolvePluginDir(runner, fs, p)
	if err != nil {
		return Paths{}, err
	}
	return p, nil
}

func absoluteDir(source, value string) (string, error) {
	if value == "" || !filepath.IsAbs(value) {
		return "", fmt.Errorf("%s must be an absolute path, got %q", source, value)
	}
	return filepath.Clean(value), nil
}

func resolveHome(source, value, fallback string) (string, error) {
	if value == "" {
		value = fallback
	}
	return absoluteDir(source, value)
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

func resolvePluginDir(runner tmux.Runner, fs FS, p Paths) (plug.Root, PathSource, error) {
	if v, err := runner.ShowOption(PluginPathOption); err == nil && strings.TrimSpace(v) != "" {
		root, rootErr := plug.NewRoot(PluginPathOption, v, p.Home, p.XDGConfigHome)
		return root, SourceOption, rootErr
	}
	if v, err := runner.ShowEnvironment(PluginPathEnvVar); err == nil && strings.TrimSpace(v) != "" {
		root, rootErr := plug.NewRoot(PluginPathEnvVar, v, p.Home, p.XDGConfigHome)
		return root, SourceEnvTpack, rootErr
	}
	if v, err := runner.ShowEnvironment(LegacyPluginPathEnvVar); err == nil && strings.TrimSpace(v) != "" {
		root, rootErr := plug.NewRoot(LegacyPluginPathEnvVar, v, p.Home, p.XDGConfigHome)
		return root, SourceEnvLegacy, rootErr
	}

	for _, cand := range orderedPluginDirs(p) {
		if fs.FileExists(cand.dir) {
			root, err := plug.NewRoot(cand.src.String(), cand.dir, p.Home, p.XDGConfigHome)
			return root, cand.src, err
		}
	}

	root, err := plug.NewRoot(SourceDefaultXDGData.String(), filepath.Join(p.XDGDataHome, "tmux", "plugins"), p.Home, p.XDGConfigHome)
	return root, SourceDefaultXDGData, err
}

type pluginDirCandidate struct {
	dir string
	src PathSource
}

func orderedPluginDirs(p Paths) []pluginDirCandidate {
	ordered := []pluginDirCandidate{
		{dir: filepath.Join(p.XDGConfigHome, "tmux", "plugins"), src: SourceDetectedXDGConfig},
		{dir: filepath.Join(p.Home, ".tmux", "plugins"), src: SourceDetectedLegacy},
	}
	if p.TmuxConf == filepath.Join(p.Home, ".tmux.conf") {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	return ordered
}

func (s PathSource) String() string {
	switch s {
	case SourceOption:
		return PluginPathOption
	case SourceEnvTpack:
		return PluginPathEnvVar
	case SourceEnvLegacy:
		return LegacyPluginPathEnvVar
	case SourceDetectedXDGConfig:
		return "existing XDG config plugin directory"
	case SourceDetectedLegacy:
		return "existing legacy plugin directory"
	case SourceDefaultXDGData:
		return "XDG data default"
	default:
		return "unknown plugin path source"
	}
}
