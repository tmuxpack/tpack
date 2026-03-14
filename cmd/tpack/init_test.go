package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
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
	fakeBinary := filepath.Join(tmpDir, "tpack")
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

func TestBindKeys_ReturnsError(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Errors["BindKey:I"] = errors.New("bind failed")
	cfg := &config.Config{
		InstallKey: "I",
		UpdateKey:  "U",
		CleanKey:   "M-u",
		TuiKey:     "T",
	}

	err := bindKeys(runner, cfg, "/usr/bin/tpack")
	if err == nil {
		t.Fatal("expected error from bindKeys when BindKey fails")
	}
	if !strings.Contains(err.Error(), "bind failed") {
		t.Errorf("expected error to contain 'bind failed', got: %v", err)
	}
}

func TestBindKeys_PopupPath(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.VersionStr = "tmux 3.4"
	cfg := &config.Config{
		InstallKey: "I",
		UpdateKey:  "U",
		CleanKey:   "M-u",
		TuiKey:     "T",
	}

	err := bindKeys(runner, cfg, "/usr/bin/tpack")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var bindCalls []tmux.Call
	for _, c := range runner.Calls {
		if c.Method == "BindKey" {
			bindCalls = append(bindCalls, c)
		}
	}

	if len(bindCalls) != 4 {
		t.Fatalf("expected 4 BindKey calls, got %d", len(bindCalls))
	}

	// All popup bindings should use display-popup with new-window fallback.
	for i, c := range bindCalls {
		cmd := c.Args[1]
		if !strings.Contains(cmd, "display-popup") {
			t.Errorf("binding %d: expected display-popup, got %s", i, cmd)
		}
		if !strings.Contains(cmd, "new-window") {
			t.Errorf("binding %d: expected new-window fallback, got %s", i, cmd)
		}
	}

	if !strings.Contains(bindCalls[0].Args[1], "tui --install") {
		t.Errorf("expected install binding to contain 'tui --install', got %s", bindCalls[0].Args[1])
	}
	if !strings.Contains(bindCalls[1].Args[1], "tui --update") {
		t.Errorf("expected update binding to contain 'tui --update', got %s", bindCalls[1].Args[1])
	}
	if !strings.Contains(bindCalls[2].Args[1], "tui --clean") {
		t.Errorf("expected clean binding to contain 'tui --clean', got %s", bindCalls[2].Args[1])
	}
	if !strings.Contains(bindCalls[3].Args[1], "tui") {
		t.Errorf("expected TUI binding to contain 'tui', got %s", bindCalls[3].Args[1])
	}
}

func TestBindKeys_InlinePath(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.VersionStr = "tmux 2.9"
	cfg := &config.Config{
		InstallKey: "I",
		UpdateKey:  "U",
		CleanKey:   "M-u",
		TuiKey:     "T",
	}

	err := bindKeys(runner, cfg, "/usr/bin/tpack")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var bindCalls []tmux.Call
	for _, c := range runner.Calls {
		if c.Method == "BindKey" {
			bindCalls = append(bindCalls, c)
		}
	}

	if len(bindCalls) != 4 {
		t.Fatalf("expected 4 BindKey calls, got %d", len(bindCalls))
	}

	for i, c := range bindCalls {
		cmd := c.Args[1]
		if !strings.Contains(cmd, "new-window") {
			t.Errorf("binding %d: expected new-window, got %s", i, cmd)
		}
		if strings.Contains(cmd, "display-popup") {
			t.Errorf("binding %d: unexpected display-popup in inline binding, got %s", i, cmd)
		}
	}

	if !strings.Contains(bindCalls[0].Args[1], "tui --install") {
		t.Errorf("expected install binding to contain 'tui --install', got %s", bindCalls[0].Args[1])
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
			binary:     "/home/user/.tmux/plugins/tpm/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       true,
		},
		{
			name:       "go install binary",
			binary:     "/home/user/go/bin/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "manual build binary",
			binary:     "/home/user/gits/tpm/dist/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "system package binary",
			binary:     "/usr/bin/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			want:       false,
		},
		{
			name:       "xdg plugin path",
			binary:     "/home/user/.config/tmux/plugins/tpm/tpack",
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
			binary:     "/home/user/.tmux/plugins/tpm/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			pinned:     "",
			want:       true,
		},
		{
			name:       "auto-downloaded, pinned",
			binary:     "/home/user/.tmux/plugins/tpm/tpack",
			pluginPath: "/home/user/.tmux/plugins/",
			pinned:     "v1.2.3",
			want:       false,
		},
		{
			name:       "not auto-downloaded",
			binary:     "/usr/bin/tpack",
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
