package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

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
			cfg := &config.Config{
				UpdateMode: "prompt",
				PluginPath: "/tmp/plugins",
			}

			result := handleOutdated(runner, cfg, nil, tt.outdated)
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
	cfg := &config.Config{
		UpdateMode: "unknown",
		PluginPath: "/tmp/plugins",
	}

	result := handleOutdated(runner, cfg, nil, []string{"tmux-sensible"})
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
	cfg := &config.Config{
		UpdateMode: "auto",
		PluginPath: t.TempDir(),
	}

	plugins := []plug.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}
	outdated := []string{"tmux-sensible"}

	// autoUpdatePlugins will attempt to update but the plugin dir doesn't
	// exist, so the manager will report nothing updated. The function should
	// still produce a DisplayMessage with either success or failure status.
	result := handleOutdated(runner, cfg, plugins, outdated)

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
