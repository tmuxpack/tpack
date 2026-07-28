package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

const tmuxConfName = "tmux.conf"

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
	// TmuxConf is the writable tmux.conf; guaranteed to exist on disk.
	TmuxConf string
	// TmuxConfs lists every active readable root in source order.
	TmuxConfs []string
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

	confs, writable, searched, err := resolveTmuxConfs(runner, fs, p)
	if err != nil {
		return Paths{}, err
	}
	if writable == "" {
		return Paths{ConfSearched: searched}, &ErrNoTmuxConf{Searched: searched}
	}
	p.TmuxConf = writable
	p.TmuxConfs = confs
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

func resolveTmuxConfs(runner tmux.Runner, fs FS, p Paths) (roots []string, writable string, searched []string, err error) {
	if value, set, showErr := runner.ShowOption(ConfigPathOption); showErr == nil && set && strings.TrimSpace(value) != "" {
		candidate := strings.TrimSpace(value)
		if !filepath.IsAbs(candidate) {
			return nil, "", nil, fmt.Errorf("%s must be an absolute existing regular file, got %q", ConfigPathOption, value)
		}
		candidate = filepath.Clean(candidate)
		if !fs.IsRegularFile(candidate) {
			return nil, "", []string{candidate}, fmt.Errorf("%s must be an absolute existing regular file, got %q", ConfigPathOption, value)
		}
		return []string{candidate}, candidate, []string{candidate}, nil
	}

	activeRoots, activeSearched, reported, activeErr := activeTmuxConfs(runner, fs, p)
	if activeErr != nil {
		return nil, "", activeSearched, activeErr
	}
	if reported {
		return activeRoots, chooseWritableTmuxConf(activeRoots, p), activeSearched, nil
	}

	roots, searched = defaultTmuxConfs(fs, p)
	return roots, chooseWritableTmuxConf(roots, p), searched, nil
}

func activeTmuxConfs(runner tmux.Runner, fs FS, p Paths) (roots, searched []string, reported bool, err error) {
	value, err := runner.ExpandFormat("#{config_files}")
	if err != nil || strings.TrimSpace(value) == "" {
		return nil, nil, false, nil
	}

	optionalDefaults := make(map[string]bool)
	for _, candidate := range defaultTmuxConfCandidates(p) {
		optionalDefaults[filepath.Clean(candidate)] = true
	}
	seen := make(map[string]bool)
	for candidate := range strings.SplitSeq(value, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			return nil, searched, true, fmt.Errorf("#{config_files} reported relative config root %q", candidate)
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		searched = append(searched, candidate)
		if !fs.IsRegularFile(candidate) {
			packagedSystemConfig := filepath.Base(candidate) == tmuxConfName && filepath.Base(filepath.Dir(candidate)) == "etc"
			if !fs.FileExists(candidate) && (optionalDefaults[candidate] || packagedSystemConfig) {
				continue
			}
			return nil, searched, true, fmt.Errorf("#{config_files} reported missing, unreadable, or non-regular config root %q", candidate)
		}
		roots = append(roots, candidate)
	}
	return roots, searched, true, nil
}

func defaultTmuxConfs(fs FS, p Paths) (roots, searched []string) {
	for _, candidate := range deduplicatePaths(defaultTmuxConfCandidates(p)) {
		searched = append(searched, candidate)
		if fs.IsRegularFile(candidate) {
			roots = append(roots, candidate)
		}
	}
	return roots, searched
}

func defaultTmuxConfCandidates(p Paths) []string {
	return []string{
		"/etc/tmux.conf",
		filepath.Join(p.Home, ".tmux.conf"),
		filepath.Join(p.XDGConfigHome, "tmux", tmuxConfName),
		filepath.Join(p.Home, ".config", "tmux", tmuxConfName),
	}
}

func deduplicatePaths(candidates []string) []string {
	seen := make(map[string]bool, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		result = append(result, candidate)
	}
	return result
}

func chooseWritableTmuxConf(roots []string, p Paths) string {
	preferences := []string{
		filepath.Join(p.XDGConfigHome, "tmux", tmuxConfName),
		filepath.Join(p.Home, ".config", "tmux", tmuxConfName),
		filepath.Join(p.Home, ".tmux.conf"),
	}
	for _, preference := range preferences {
		for _, root := range roots {
			if root == preference {
				return root
			}
		}
	}
	for i := len(roots) - 1; i >= 0; i-- {
		if roots[i] != "/etc/tmux.conf" {
			return roots[i]
		}
	}
	return ""
}

func resolvePluginDir(runner tmux.Runner, fs FS, p Paths) (plug.Root, PathSource, error) {
	if v, set, err := runner.ShowOption(PluginPathOption); err == nil && set {
		root, rootErr := plug.NewRoot(PluginPathOption, v, p.Home, p.XDGConfigHome)
		return root, SourceOption, rootErr
	}
	if v, err := runner.ShowEnvironment(PluginPathEnvVar); err == nil {
		root, rootErr := plug.NewRoot(PluginPathEnvVar, v, p.Home, p.XDGConfigHome)
		return root, SourceEnvTpack, rootErr
	}
	if v, err := runner.ShowEnvironment(LegacyPluginPathEnvVar); err == nil {
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
