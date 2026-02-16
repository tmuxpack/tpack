package e2e_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, pluginDir, "tmux-plugins/tmux-example-plugin")

	startTmux(t, home, socket)

	// No args: expect exit code 1 and usage message.
	output, exitCode := runInTmux(t, home, socket, binary+" update", 10*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for no-args update, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, "usage")

	// Update a single plugin by name.
	output, exitCode = runInTmux(t, home, socket, binary+" update tmux-example-plugin", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for single plugin update, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" update success`)

	// Update all plugins.
	output, exitCode = runInTmux(t, home, socket, binary+" update all", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for update all, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" update success`)
}
