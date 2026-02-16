package e2e_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestInstallLegacySyntaxViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @tpm_plugins \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	// First install: expect download success.
	output, exitCode := runInTmux(t, home, socket, binary+" install", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" download success`)

	pluginDir := filepath.Join(home, ".tmux", "plugins", "tmux-example-plugin")
	assertDirExists(t, pluginDir)

	// Second install: expect already installed.
	output, _ = runInTmux(t, home, socket, binary+" install", 60*time.Second)
	assertContains(t, output, `Already installed "tmux-example-plugin"`)
}

func TestInstallLegacyAndNewSyntaxViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @tpm_plugins \"tmux-plugins/tmux-example-plugin\"\nset -g @plugin \"tmux-plugins/tmux-copycat\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	// First install: expect both plugins downloaded.
	output, exitCode := runInTmux(t, home, socket, binary+" install", 120*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}

	exampleDir := filepath.Join(home, ".tmux", "plugins", "tmux-example-plugin")
	copycatDir := filepath.Join(home, ".tmux", "plugins", "tmux-copycat")
	assertDirExists(t, exampleDir)
	assertDirExists(t, copycatDir)

	// Second install: expect already installed.
	output, _ = runInTmux(t, home, socket, binary+" install", 120*time.Second)
	assertContains(t, output, `Already installed "tmux-copycat"`)
}

func TestInitBindsKeysWithLegacySyntax(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @tpm_plugins \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	expectedPath := filepath.Join(home, ".tmux", "plugins") + "/"

	startTmux(t, home, socket)

	// Wait for init to complete by checking TMUX_PLUGIN_MANAGER_PATH.
	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	keys := tmuxListKeys(t, socket)

	assertContains(t, keys, "tui --popup --install")
}

func TestInitWithLegacyAndNewSyntax(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @tpm_plugins \"tmux-plugins/tmux-example-plugin\"\nset -g @plugin \"tmux-plugins/tmux-copycat\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	expectedPath := filepath.Join(home, ".tmux", "plugins") + "/"

	startTmux(t, home, socket)

	// Wait for init to complete by checking TMUX_PLUGIN_MANAGER_PATH.
	waitForEnv(t, socket, "TMUX_PLUGIN_MANAGER_PATH", expectedPath, 15)

	keys := tmuxListKeys(t, socket)

	assertContains(t, keys, "tui --popup --install")
}
