package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	// Empty tmux.conf: no plugins declared, so any installed plugin should be cleaned.
	tmuxConf := fmt.Sprintf("run-shell \"%s\"\n", binary)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, pluginDir, "tmux-plugins/tmux-example-plugin")

	exampleDir := filepath.Join(pluginDir, "tmux-example-plugin")
	assertDirExists(t, exampleDir)

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" clean success`)
	assertDirNotExists(t, exampleDir)
}

func TestCleanFailsOnPermissionDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	if os.Getuid() == 0 {
		t.Skip("skipping permission-denied test when running as root")
	}

	binary := buildBinary(t)

	// Empty tmux.conf: no plugins declared, so any installed plugin should be cleaned.
	tmuxConf := fmt.Sprintf("run-shell \"%s\"\n", binary)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, pluginDir, "tmux-plugins/tmux-example-plugin")

	exampleDir := filepath.Join(pluginDir, "tmux-example-plugin")
	assertDirExists(t, exampleDir)

	// Remove all permissions to prevent deletion.
	if err := os.Chmod(exampleDir, 0o000); err != nil {
		t.Fatalf("failed to chmod plugin directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(exampleDir, 0o755)
	})

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" clean fail`)
}
