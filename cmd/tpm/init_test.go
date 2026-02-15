package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestFindBinary(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		result := findBinary()
		if result == "" {
			t.Fatal("findBinary() returned empty string")
		}
	})

	t.Run("returned path exists on disk", func(t *testing.T) {
		result := findBinary()
		if _, err := os.Stat(result); err != nil {
			t.Errorf("findBinary() returned %q which does not exist: %v", result, err)
		}
	})
}

func TestFindBinary_WithCurrentDirEnv(t *testing.T) {
	// In test context os.Executable succeeds, so findBinary always returns
	// the test binary path regardless of CURRENT_DIR. We verify that
	// findBinary still returns a valid, existing path when CURRENT_DIR is set.
	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "tpm-go")
	if err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CURRENT_DIR", tmpDir)

	result := findBinary()
	if result == "" {
		t.Fatal("findBinary() returned empty string with CURRENT_DIR set")
	}
	if _, err := os.Stat(result); err != nil {
		t.Errorf("findBinary() returned %q which does not exist: %v", result, err)
	}
}

func TestBindKeys_UseTuiPopup(t *testing.T) {
	runner := tmux.NewMockRunner()
	cfg := &config.Config{
		InstallKey: "I",
		UpdateKey:  "U",
		CleanKey:   "M-u",
		TuiKey:     "T",
	}

	bindKeys(runner, cfg, "/usr/bin/tpm-go")

	if len(runner.Calls) != 4 {
		t.Fatalf("expected 4 BindKey calls, got %d", len(runner.Calls))
	}

	// Verify install key binding uses tui --popup --install.
	installCall := runner.Calls[0]
	if installCall.Method != "BindKey" {
		t.Errorf("expected BindKey method, got %s", installCall.Method)
	}
	if !strings.Contains(installCall.Args[1], "tui --popup --install") {
		t.Errorf("expected install binding to use 'tui --popup --install', got %s", installCall.Args[1])
	}

	// Verify update key binding uses tui --popup --update.
	updateCall := runner.Calls[1]
	if !strings.Contains(updateCall.Args[1], "tui --popup --update") {
		t.Errorf("expected update binding to use 'tui --popup --update', got %s", updateCall.Args[1])
	}

	// Verify clean key binding uses tui --popup --clean.
	cleanCall := runner.Calls[2]
	if !strings.Contains(cleanCall.Args[1], "tui --popup --clean") {
		t.Errorf("expected clean binding to use 'tui --popup --clean', got %s", cleanCall.Args[1])
	}

	// Verify TUI key binding uses tui --popup.
	tuiCall := runner.Calls[3]
	if !strings.Contains(tuiCall.Args[1], "tui --popup") {
		t.Errorf("expected tui binding to use 'tui --popup', got %s", tuiCall.Args[1])
	}
}

func TestIsAutoDownloaded(t *testing.T) {
	tests := []struct {
		name       string
		binary     string
		pluginPath string
		want       bool
	}{
		{
			name:       "auto-downloaded binary",
			binary:     "/home/user/.tmux/plugins/tpm/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       true,
		},
		{
			name:       "go install binary",
			binary:     "/home/user/go/bin/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "manual build binary",
			binary:     "/home/user/gits/tpm/dist/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "system package binary",
			binary:     "/usr/bin/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "xdg plugin path",
			binary:     "/home/user/.config/tmux/plugins/tpm/tpm-go",
			pluginPath: "/home/user/.config/tmux/plugins/",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAutoDownloaded(tt.binary, tt.pluginPath)
			if got != tt.want {
				t.Errorf("isAutoDownloaded(%q, %q) = %v, want %v", tt.binary, tt.pluginPath, got, tt.want)
			}
		})
	}
}

func TestShouldSpawnSelfUpdate(t *testing.T) {
	tests := []struct {
		name       string
		binary     string
		pluginPath string
		pinned     string
		want       bool
	}{
		{
			name:       "auto-downloaded, not pinned",
			binary:     "/home/user/.tmux/plugins/tpm/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			pinned:     "",
			want:       true,
		},
		{
			name:       "auto-downloaded, pinned",
			binary:     "/home/user/.tmux/plugins/tpm/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			pinned:     "v1.2.3",
			want:       false,
		},
		{
			name:       "not auto-downloaded",
			binary:     "/usr/bin/tpm-go",
			pluginPath: "/home/user/.tmux/plugins/",
			pinned:     "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSpawnSelfUpdate(tt.binary, tt.pluginPath, tt.pinned)
			if got != tt.want {
				t.Errorf("shouldSpawnSelfUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldSpawnUpdateCheck(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		interval time.Duration
		want     bool
	}{
		{"prompt with interval", "prompt", 24 * time.Hour, true},
		{"auto with interval", "auto", 1 * time.Hour, true},
		{"off mode", "off", 24 * time.Hour, false},
		{"empty mode", "", 24 * time.Hour, false},
		{"zero interval", "prompt", 0, false},
		{"negative interval", "prompt", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				UpdateMode:          tt.mode,
				UpdateCheckInterval: tt.interval,
			}
			got := shouldSpawnUpdateCheck(cfg)
			if got != tt.want {
				t.Errorf("shouldSpawnUpdateCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
