package e2e_test

import (
	"context"
	"fmt"
	"os/exec"
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
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-example-plugin")

	startTmux(t, home, socket)

	// No args: should update all plugins (same as "update all").
	output, exitCode := runInTmux(t, home, socket, binary+" update", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for no-args update, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-plugins/tmux-example-plugin" update success`)

	// Update a single plugin by name.
	output, exitCode = runInTmux(t, home, socket, binary+" update tmux-plugins/tmux-example-plugin", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for single plugin update, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-plugins/tmux-example-plugin" update success`)

	// Update all plugins.
	output, exitCode = runInTmux(t, home, socket, binary+" update all", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for update all, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-plugins/tmux-example-plugin" update success`)
}

func TestUpdateCompletionDoesNotMigrateLegacyDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoGit(t)

	binary := buildBinary(t)
	raw := "tmux-plugins/tmux-example-plugin"
	home, _ := e2eEnv(t, "set -g @plugin \""+raw+"\"\n")
	pluginRoot := filepath.Join(home, ".tmux", "plugins")
	legacyDir := prepareLegacyRepository(t, pluginRoot, raw)
	canonicalDir := canonicalPluginDir(t, pluginRoot, raw)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "__complete", "update", "")
	cmd.Env = cleanEnv(home)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("completion failed: %v\n%s", err, out)
	}

	assertDirExists(t, legacyDir)
	assertDirNotExists(t, canonicalDir)
}
