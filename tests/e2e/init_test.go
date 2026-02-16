package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitBindsKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	expectedPath := filepath.Join(home, ".tmux", "plugins") + "/"

	startTmux(t, home, socket)

	// Wait for init to complete by checking TMUX_PLUGIN_MANAGER_PATH.
	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	keys := tmuxListKeys(t, socket)

	assertContains(t, keys, "tui --popup --install")
	assertContains(t, keys, "tui --popup --update")
	assertContains(t, keys, "tui --popup --clean")

	// The TUI key binding uses "tui --popup" without a sub-command.
	// Count occurrences of "tui --popup" vs the specific sub-commands.
	// There should be at least 4 total: install, update, clean, and standalone.
	totalPopup := strings.Count(keys, "tui --popup")
	subCommands := strings.Count(keys, "tui --popup --install") +
		strings.Count(keys, "tui --popup --update") +
		strings.Count(keys, "tui --popup --clean")

	if totalPopup <= subCommands {
		t.Errorf("expected standalone 'tui --popup' binding in addition to sub-command bindings; "+
			"total 'tui --popup' occurrences: %d, sub-command occurrences: %d",
			totalPopup, subCommands)
	}
}

func TestInitBindsKeysSetOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	// Use set-option instead of set (both are valid tmux syntax).
	tmuxConf := fmt.Sprintf(
		"set-option -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	expectedPath := filepath.Join(home, ".tmux", "plugins") + "/"

	startTmux(t, home, socket)

	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	keys := tmuxListKeys(t, socket)

	assertContains(t, keys, "tui --popup --install")
}

func TestInitCustomPluginDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	customDir := filepath.Join(t.TempDir(), "custom", "plugins")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("failed to create custom plugin dir: %v", err)
	}

	expectedPath := customDir + "/"

	tmuxConf := fmt.Sprintf(
		"set-environment -g TMUX_PLUGIN_MANAGER_PATH '%s'\nset -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
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
