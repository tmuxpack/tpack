package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestResolvePathsWithMissingHomebrewSystemConfig(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Homebrew tmux config discovery is macOS-specific")
	}

	brew, err := exec.LookPath("brew")
	requireHomebrewConditionf(t, err == nil, "brew is unavailable: %v", err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	brewPrefixCommand := exec.CommandContext(ctx, brew, "--prefix")
	brewPrefixOutput, err := brewPrefixCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("brew --prefix: %v\n%s", err, brewPrefixOutput)
	}
	brewPrefix := strings.TrimSpace(string(brewPrefixOutput))

	tmuxPrefixCommand := exec.CommandContext(ctx, brew, "--prefix", "tmux")
	tmuxPrefixOutput, err := tmuxPrefixCommand.CombinedOutput()
	requireHomebrewConditionf(t, err == nil, "brew --prefix tmux: %v\n%s", err, tmuxPrefixOutput)
	tmuxBin := filepath.Join(strings.TrimSpace(string(tmuxPrefixOutput)), "bin", "tmux")

	systemConfig := filepath.Join(brewPrefix, "etc", "tmux.conf")
	if _, statErr := os.Stat(systemConfig); statErr == nil {
		requireHomebrewConditionf(t, false, "Homebrew system config exists; missing-file precondition not met: %s", systemConfig)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("stat Homebrew system config %s: %v", systemConfig, statErr)
	}

	home := t.TempDir()
	xdgConfigHome := filepath.Join(home, "xdg")
	xdgConfig := filepath.Join(xdgConfigHome, "tmux", "tmux.conf")
	if mkdirErr := os.MkdirAll(filepath.Dir(xdgConfig), 0o755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	if writeErr := os.WriteFile(xdgConfig, nil, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	t.Setenv("PATH", filepath.Dir(tmuxBin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	serverEnv := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || name == "TMUX" || name == "TMUX_PANE" {
			continue
		}
		serverEnv = append(serverEnv, entry)
	}

	socket := fmt.Sprintf("tpack-brew-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		kill := exec.CommandContext(cleanupCtx, tmuxBin, "-L", socket, "kill-server")
		kill.Env = serverEnv
		_ = kill.Run()
	})
	start := exec.CommandContext(ctx, tmuxBin, "-L", socket, "new-session", "-d")
	start.Env = serverEnv
	if output, startErr := start.CombinedOutput(); startErr != nil {
		t.Fatalf("start Homebrew tmux: %v\n%s", startErr, output)
	}

	format := exec.CommandContext(ctx, tmuxBin, "-L", socket, "display-message", "-p", "#{config_files}")
	format.Env = serverEnv
	formatOutput, err := format.CombinedOutput()
	if err != nil {
		t.Fatalf("read config_files: %v\n%s", err, formatOutput)
	}
	configFiles := strings.TrimSpace(string(formatOutput))
	containsConfig := func(path string) bool {
		for candidate := range strings.SplitSeq(configFiles, ",") {
			if filepath.Clean(strings.TrimSpace(candidate)) == path {
				return true
			}
		}
		return false
	}
	requireHomebrewConditionf(t, containsConfig(systemConfig),
		"Homebrew tmux did not report system config %s; config_files=%q", systemConfig, configFiles)
	if !containsConfig(xdgConfig) {
		t.Fatalf("tmux did not report XDG config %s; config_files=%q", xdgConfig, configFiles)
	}

	tmuxValue := exec.CommandContext(ctx, tmuxBin, "-L", socket, "display-message", "-p", "#{socket_path},#{pid},#{session_id}")
	tmuxValue.Env = serverEnv
	tmuxOutput, err := tmuxValue.CombinedOutput()
	if err != nil {
		t.Fatalf("read TMUX value: %v\n%s", err, tmuxOutput)
	}
	t.Setenv("TMUX", strings.TrimSpace(string(tmuxOutput)))
	paneValue := exec.CommandContext(ctx, tmuxBin, "-L", socket, "display-message", "-p", "#{pane_id}")
	paneValue.Env = serverEnv
	paneOutput, err := paneValue.CombinedOutput()
	if err != nil {
		t.Fatalf("read TMUX_PANE value: %v\n%s", err, paneOutput)
	}
	t.Setenv("TMUX_PANE", strings.TrimSpace(string(paneOutput)))

	paths, err := config.ResolvePaths(tmux.NewRealRunner(), config.RealFS{}, config.Env{
		Home:          home,
		XDGConfigHome: xdgConfigHome,
	})
	if err != nil {
		t.Fatalf("ResolvePaths with config_files %q: %v", configFiles, err)
	}
	if want := []string{xdgConfig}; !reflect.DeepEqual(paths.TmuxConfs, want) {
		t.Fatalf("TmuxConfs = %v, want %v; config_files=%q", paths.TmuxConfs, want, configFiles)
	}
	if paths.TmuxConf != xdgConfig {
		t.Fatalf("TmuxConf = %q, want %q", paths.TmuxConf, xdgConfig)
	}
	if !slices.Contains(paths.ConfSearched, systemConfig) {
		t.Fatalf("ConfSearched = %v, want Homebrew system config %q; config_files=%q", paths.ConfSearched, systemConfig, configFiles)
	}
}

func requireHomebrewConditionf(t *testing.T, condition bool, format string, args ...any) {
	t.Helper()
	if condition {
		return
	}
	if os.Getenv("CI") != "" {
		t.Fatalf(format, args...)
	}
	t.Skipf(format, args...)
}
