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

	// One plugin declared; an additional unlisted plugin is an orphan that
	// clean should remove while leaving the declared plugin alone.
	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-example-plugin")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")

	exampleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-example-plugin")
	orphanPlugin := mustParsePlugin(t, "tmux-plugins/tmux-sensible")
	orphanDir := filepath.Join(pluginDir, orphanPlugin.DirName)
	assertDirExists(t, exampleDir)
	assertDirExists(t, orphanDir)

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"`+orphanPlugin.DirName+`" clean success`)
	assertDirNotExists(t, orphanDir)
	assertDirExists(t, exampleDir)
}

func TestCleanWithEmptyConfigRemovesAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	// Empty tmux.conf: no plugins declared, but the config is readable.
	// Clean now treats every installed plugin as an orphan, matching TPM's
	// original contract (a missing/unreadable config is a separate, explicit
	// error case, not this one).
	tmuxConf := fmt.Sprintf("run-shell \"%s\"\n", binary)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-example-plugin")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")

	exampleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-example-plugin")
	sensibleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-sensible")
	assertDirExists(t, exampleDir)
	assertDirExists(t, sensibleDir)

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertDirNotExists(t, exampleDir)
	assertDirNotExists(t, sensibleDir)
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

	// One plugin declared; an additional unlisted plugin is the orphan that
	// clean will attempt (and fail) to remove.
	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-example-plugin")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")

	exampleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-example-plugin")
	orphanPlugin := mustParsePlugin(t, "tmux-plugins/tmux-sensible")
	orphanDir := filepath.Join(pluginDir, orphanPlugin.DirName)
	assertDirExists(t, exampleDir)
	assertDirExists(t, orphanDir)

	// Remove all permissions on the orphan to prevent deletion.
	if err := os.Chmod(orphanDir, 0o000); err != nil {
		t.Fatalf("failed to chmod plugin directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(orphanDir, 0o755)
	})

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"`+orphanPlugin.DirName+`" clean fail`)
	assertDirExists(t, exampleDir)
}

func TestCleanPreservesPluginsWhenRequiredSourceIsMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)
	binary := buildBinary(t)
	home, socket := e2eEnv(t, "")
	startTmux(t, home, socket)
	confPath := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(confPath, []byte("source ~/.tmux/plugins.conf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-example-plugin")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")
	exampleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-example-plugin")
	sensibleDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-sensible")
	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, "cannot read required source")
	assertDirExists(t, exampleDir)
	assertDirExists(t, sensibleDir)
}

func TestCleanAllowsMissingQuietSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)
	binary := buildBinary(t)
	home, socket := e2eEnv(t, "source-file -q ~/.tmux/plugins.conf\n")
	startTmux(t, home, socket)
	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")
	orphanDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-sensible")
	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertDirNotExists(t, orphanDir)
}

func TestCleanPreservesPluginsWhenNestedSourceCannotBeEvaluated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)
	binary := buildBinary(t)
	home, socket := e2eEnv(t, "")
	startTmux(t, home, socket)

	confPath := filepath.Join(home, ".tmux.conf")
	content := "if-shell 'test -f ~/.tmux/plugins.conf' 'source-file ~/.tmux/plugins.conf'\n"
	if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".tmux", "plugins.conf"),
		[]byte("set -g @plugin tmux-plugins/tmux-sensible\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	installPluginManually(t, home, pluginDir, "tmux-plugins/tmux-sensible")
	installedDir := canonicalPluginDir(t, pluginDir, "tmux-plugins/tmux-sensible")

	output, exitCode := runInTmux(t, home, socket, binary+" clean", 30*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, "executable quoted command list may contain source")
	assertDirExists(t, installedDir)
}
