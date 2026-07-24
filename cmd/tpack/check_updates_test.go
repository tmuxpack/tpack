package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

func mustRoot(t *testing.T, path string) plug.Root {
	t.Helper()
	root, err := plug.NewRoot("test", path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestUpdateChecksEnabled(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		interval time.Duration
		want     bool
	}{
		{
			name:     "prompt mode with positive interval",
			mode:     "prompt",
			interval: 24 * time.Hour,
			want:     true,
		},
		{
			name:     "auto mode with positive interval",
			mode:     "auto",
			interval: 1 * time.Hour,
			want:     true,
		},
		{
			name:     "off mode with positive interval",
			mode:     "off",
			interval: 24 * time.Hour,
			want:     false,
		},
		{
			name:     "empty mode with positive interval",
			mode:     "",
			interval: 24 * time.Hour,
			want:     false,
		},
		{
			name:     "prompt mode with zero interval",
			mode:     "prompt",
			interval: 0,
			want:     false,
		},
		{
			name:     "prompt mode with negative interval",
			mode:     "prompt",
			interval: -1 * time.Second,
			want:     false,
		},
		{
			name:     "auto mode with zero interval",
			mode:     "auto",
			interval: 0,
			want:     false,
		},
		{
			name:     "off mode with zero interval",
			mode:     "off",
			interval: 0,
			want:     false,
		},
		{
			name:     "empty mode with zero interval",
			mode:     "",
			interval: 0,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				UpdateMode:          tt.mode,
				UpdateCheckInterval: tt.interval,
			}
			got := updateChecksEnabled(cfg)
			if got != tt.want {
				t.Errorf("updateChecksEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindOutdatedPluginsUsesDirNameAndReturnsName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	rootPath := t.TempDir()
	p, err := plug.ParseSpec("catppuccin/tmux", nil)
	if err != nil {
		t.Fatal(err)
	}
	bare := initOutdatedRepo(t)
	pluginDir := filepath.Join(rootPath, "tmux-87a1216f1f68")
	runGitCommand(t, "", "clone", bare, pluginDir)
	advanceOutdatedRepo(t, bare)

	got := findOutdatedPlugins([]plug.Plugin{p}, mustRoot(t, rootPath))
	if len(got) != 1 || got[0] != "tmux" {
		t.Fatalf("outdated plugins = %v, want [tmux]", got)
	}
}

func initOutdatedRepo(t *testing.T) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "remote.git")
	runGitCommand(t, "", "init", "--bare", bare)
	work := filepath.Join(t.TempDir(), "work")
	runGitCommand(t, "", "clone", bare, work)
	runGitCommand(t, work, "config", "user.email", "test@example.com")
	runGitCommand(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "README"), []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, work, "add", "README")
	runGitCommand(t, work, "commit", "-m", "initial")
	runGitCommand(t, work, "push", "origin", "HEAD")
	return bare
}

func advanceOutdatedRepo(t *testing.T, bare string) {
	t.Helper()
	work := filepath.Join(t.TempDir(), "advance")
	runGitCommand(t, "", "clone", bare, work)
	runGitCommand(t, work, "config", "user.email", "test@example.com")
	runGitCommand(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "update"), []byte("update"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, work, "add", "update")
	runGitCommand(t, work, "commit", "-m", "update")
	runGitCommand(t, work, "push", "origin", "HEAD")
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestHandleOutdated_PromptMode(t *testing.T) {
	tests := []struct {
		name         string
		outdated     []string
		wantContains string
	}{
		{
			name:         "single outdated plugin",
			outdated:     []string{"tmux-sensible"},
			wantContains: "1 plugin update(s) available",
		},
		{
			name:         "multiple outdated plugins",
			outdated:     []string{"tmux-sensible", "tmux-resurrect", "tmux-continuum"},
			wantContains: "3 plugin update(s) available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tmux.NewMockRunner()
			status := ui.NewStatusOutput(runner)
			cfg := &config.Config{
				UpdateMode: "prompt",
				PluginPath: mustRoot(t, "/tmp/plugins"),
			}

			result := handleOutdated(cfg, nil, tt.outdated, ui.NewMockOutput(), status)
			if result != 0 {
				t.Errorf("handleOutdated() = %d, want 0", result)
			}

			// Verify exactly one DisplayMessage call with the correct content.
			var displayCalls []tmux.Call
			for _, call := range runner.Calls {
				if call.Method == "DisplayMessage" {
					displayCalls = append(displayCalls, call)
				}
			}

			if len(displayCalls) != 1 {
				t.Fatalf("expected 1 DisplayMessage call, got %d", len(displayCalls))
			}

			msg := displayCalls[0].Args[0]
			wantMsg := "tpack: " + strconv.Itoa(len(tt.outdated)) + " plugin update(s) available. Press prefix+U to update."
			if msg != wantMsg {
				t.Errorf("DisplayMessage = %q, want %q", msg, wantMsg)
			}
		})
	}
}

func TestHandleOutdated_UnknownMode(t *testing.T) {
	runner := tmux.NewMockRunner()
	status := ui.NewStatusOutput(runner)
	cfg := &config.Config{
		UpdateMode: "unknown",
		PluginPath: mustRoot(t, "/tmp/plugins"),
	}

	result := handleOutdated(cfg, nil, []string{"tmux-sensible"}, ui.NewMockOutput(), status)
	if result != 0 {
		t.Errorf("handleOutdated() = %d, want 0 for unrecognized mode", result)
	}

	// No DisplayMessage should be called for an unrecognized mode.
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" {
			t.Errorf("unexpected DisplayMessage call for unrecognized mode: %v", call.Args)
		}
	}
}

func TestHandleOutdated_AutoMode(t *testing.T) {
	// autoUpdatePlugins calls newManagerDeps which needs a real plugin path
	// and creates a Manager. We verify the function is invoked by checking
	// that a DisplayMessage is produced (either success or failure message).
	runner := tmux.NewMockRunner()
	status := ui.NewStatusOutput(runner)
	cfg := &config.Config{
		UpdateMode: "auto",
		PluginPath: mustRoot(t, t.TempDir()),
	}

	plugins := []plug.Plugin{
		{Name: "tmux-sensible", DirName: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}
	outdated := []string{"tmux-sensible"}

	// autoUpdatePlugins will attempt to update but the plugin dir doesn't
	// exist, so the manager will report nothing updated. The function should
	// still produce a DisplayMessage with either success or failure status.
	result := handleOutdated(cfg, plugins, outdated, ui.NewMockOutput(), status)

	var displayCalls []tmux.Call
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" {
			displayCalls = append(displayCalls, call)
		}
	}

	if len(displayCalls) == 0 {
		t.Fatal("expected at least one DisplayMessage call for auto mode")
	}

	// The result should be 0 (success) or 1 (failure) depending on whether
	// the update succeeded. Either way, a message should have been displayed.
	if result != 0 && result != 1 {
		t.Errorf("handleOutdated() = %d, want 0 or 1", result)
	}
}

func TestHandleOutdated_PromptTransportFailure(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Errors["DisplayMessage"] = errors.New("tmux unavailable")
	status := ui.NewStatusOutput(runner)
	cfg := &config.Config{UpdateMode: updateModePrompt}

	if got := handleOutdated(cfg, nil, []string{"tmux-sensible"}, ui.NewMockOutput(), status); got != 1 {
		t.Fatalf("handleOutdated() = %d, want 1", got)
	}
	var transport *ui.TransportError
	if !errors.As(status.Result(), &transport) {
		t.Fatalf("status result = %v, want transport error", status.Result())
	}
}

func TestHandleOutdated_AutoTransportFailure(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Errors["DisplayMessage"] = errors.New("tmux unavailable")
	status := ui.NewStatusOutput(runner)
	cfg := &config.Config{
		UpdateMode: updateModeAuto,
		PluginPath: mustRoot(t, t.TempDir()),
	}

	if got := handleOutdated(cfg, nil, []string{"all"}, ui.NewMockOutput(), status); got != 1 {
		t.Fatalf("handleOutdated() = %d, want 1", got)
	}
	var transport *ui.TransportError
	if !errors.As(status.Result(), &transport) {
		t.Fatalf("status result = %v, want transport error", status.Result())
	}
}

func TestCheckUpdatesResultReturnsTransportFailure(t *testing.T) {
	sink := ui.NewMockSink()
	sink.Err = errors.New("tmux unavailable")
	output := ui.NewReporter(sink)
	output.Ok("updates available")

	err := checkUpdatesResult(0, output)
	var transport *ui.TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("checkUpdatesResult() = %v, want transport error", err)
	}
}

func TestCheckUpdatesResultUsesErrSilentForDeliveredFailure(t *testing.T) {
	output := ui.NewReporter(ui.NewMockSink())
	output.Err("config failed")

	if err := checkUpdatesResult(1, output); !errors.Is(err, errSilent) {
		t.Fatalf("checkUpdatesResult() = %v, want errSilent", err)
	}
}
