package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	}

	bindKeys(runner, cfg)

	if len(runner.Calls) != 3 {
		t.Fatalf("expected 3 BindKey calls, got %d", len(runner.Calls))
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
}
