package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginSourcing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	// Use a separate temp dir for the marker so it is outside HOME.
	markerDir := t.TempDir()
	markerFile := filepath.Join(markerDir, "plugin_sourced")

	// Set up an isolated HOME with a minimal tmux.conf (e2eEnv writes one).
	home, socket := e2eEnv(t, "# placeholder")

	// Create the fake plugin directory and executable .tmux file.
	pluginDir := filepath.Join(home, ".tmux", "plugins", "tmux_test_plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	tmuxScript := fmt.Sprintf("#!/usr/bin/env bash\ntouch %s\n", markerFile)
	scriptPath := filepath.Join(pluginDir, "test_plugin.tmux")
	if err := os.WriteFile(scriptPath, []byte(tmuxScript), 0o755); err != nil {
		t.Fatalf("failed to write test_plugin.tmux: %v", err)
	}

	// Overwrite .tmux.conf after e2eEnv so it references the plugin and binary.
	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"doesnt_matter/tmux_test_plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	confPath := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(confPath, []byte(tmuxConf), 0o644); err != nil {
		t.Fatalf("failed to overwrite tmux.conf: %v", err)
	}

	startTmux(t, home, socket)

	waitForFile(t, markerFile, 15)

	if _, err := os.Stat(markerFile); err != nil {
		t.Errorf("expected marker file %q to exist after plugin sourcing, got error: %v", markerFile, err)
	}
}

func TestDefaultTpmPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf("run-shell \"%s\"\n", binary)
	home, socket := e2eEnv(t, tmuxConf)

	expectedPath := filepath.Join(home, ".tmux", "plugins") + "/"

	startTmux(t, home, socket)

	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	got := tmuxShowEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH")
	if got != expectedPath {
		t.Errorf("TMUX_PLUGIN_MANAGER_PATH = %q, want %q", got, expectedPath)
	}
}

func TestCustomTpmPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	customDir := filepath.Join(t.TempDir(), "custom", "plugins")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("failed to create custom plugin dir: %v", err)
	}

	// resolvePluginPath appends a trailing slash, so expect it in the result.
	expectedPath := customDir + "/"

	tmuxConf := fmt.Sprintf(
		"set-environment -g TMUX_PLUGIN_MANAGER_PATH '%s'\nrun-shell \"%s\"\n",
		customDir, binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	startTmux(t, home, socket)

	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	got := tmuxShowEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH")
	if got != expectedPath {
		t.Errorf("TMUX_PLUGIN_MANAGER_PATH = %q, want %q", got, expectedPath)
	}
}
