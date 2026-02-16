package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstallViaCLI(t *testing.T) {
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

func TestInstallCustomDirViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	customDir := filepath.Join(t.TempDir(), "custom", "plugins")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("failed to create custom plugin dir: %v", err)
	}

	// Trailing slash is required by the binary.
	tmuxConf := fmt.Sprintf(
		"set-environment -g TMUX_PLUGIN_MANAGER_PATH '%s/'\nset -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		customDir, binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	// First install: expect download success.
	output, exitCode := runInTmux(t, home, socket, binary+" install", 60*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"tmux-example-plugin" download success`)

	pluginDir := filepath.Join(customDir, "tmux-example-plugin")
	assertDirExists(t, pluginDir)

	// Second install: expect already installed.
	output, _ = runInTmux(t, home, socket, binary+" install", 60*time.Second)
	assertContains(t, output, `Already installed "tmux-example-plugin"`)
}

func TestInstallNonExistentViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/non-existing-plugin\"\nrun-shell \"%s\"\n",
		binary,
	)
	home, socket := e2eEnv(t, tmuxConf)
	startTmux(t, home, socket)

	output, exitCode := runInTmux(t, home, socket, binary+" install", 60*time.Second)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput: %s", exitCode, output)
	}
	assertContains(t, output, `"non-existing-plugin" download fail`)
}

func TestInstallMultipleViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	tmuxConf := fmt.Sprintf(
		"set -g @plugin \"tmux-plugins/tmux-example-plugin\"\nset -g @plugin \"tmux-plugins/tmux-copycat\"\nrun-shell \"%s\"\n",
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

	// Second install: expect already installed messages.
	output, _ = runInTmux(t, home, socket, binary+" install", 120*time.Second)
	assertContains(t, output, `Already installed "tmux-example-plugin"`)
}

func TestInstallFromSourcedFileViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	// Create the isolated environment first with a placeholder config.
	home, socket := e2eEnv(t, "# placeholder")

	// Write the additional config file in $HOME/.tmux/ (directory already exists from e2eEnv).
	additionalConf := filepath.Join(home, ".tmux", "additional_config_file_1")
	if err := os.WriteFile(additionalConf, []byte("set -g @plugin 'tmux-plugins/tmux-copycat'\n"), 0o644); err != nil {
		t.Fatalf("failed to write additional config: %v", err)
	}

	// Overwrite .tmux.conf to source the additional file and declare main plugin.
	tmuxConf := fmt.Sprintf(
		"source '%s'\nset -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		additionalConf, binary,
	)
	confPath := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(confPath, []byte(tmuxConf), 0o644); err != nil {
		t.Fatalf("failed to overwrite tmux.conf: %v", err)
	}

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

func TestInstallFromMultipleSourcedFilesViaCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	skipIfNoTmux(t)
	skipIfNoGit(t)

	binary := buildBinary(t)

	// Create the isolated environment first with a placeholder config.
	home, socket := e2eEnv(t, "# placeholder")

	// Write the additional config files in $HOME/.tmux/ (directory already exists from e2eEnv).
	additionalConf1 := filepath.Join(home, ".tmux", "additional_config_file_1")
	if err := os.WriteFile(additionalConf1, []byte("set -g @plugin 'tmux-plugins/tmux-copycat'\n"), 0o644); err != nil {
		t.Fatalf("failed to write additional config 1: %v", err)
	}

	additionalConf2 := filepath.Join(home, ".tmux", "additional_config_file_2")
	if err := os.WriteFile(additionalConf2, []byte("set -g @plugin 'tmux-plugins/tmux-sensible'\n"), 0o644); err != nil {
		t.Fatalf("failed to write additional config 2: %v", err)
	}

	// Overwrite .tmux.conf to source both files and declare main plugin.
	tmuxConf := fmt.Sprintf(
		"source '%s'\nsource-file '%s'\nset -g @plugin \"tmux-plugins/tmux-example-plugin\"\nrun-shell \"%s\"\n",
		additionalConf1, additionalConf2, binary,
	)
	confPath := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(confPath, []byte(tmuxConf), 0o644); err != nil {
		t.Fatalf("failed to overwrite tmux.conf: %v", err)
	}

	startTmux(t, home, socket)

	// First install: expect all three plugins downloaded.
	output, exitCode := runInTmux(t, home, socket, binary+" install", 120*time.Second)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", exitCode, output)
	}

	exampleDir := filepath.Join(home, ".tmux", "plugins", "tmux-example-plugin")
	copycatDir := filepath.Join(home, ".tmux", "plugins", "tmux-copycat")
	sensibleDir := filepath.Join(home, ".tmux", "plugins", "tmux-sensible")
	assertDirExists(t, exampleDir)
	assertDirExists(t, copycatDir)
	assertDirExists(t, sensibleDir)

	// Second install: expect already installed.
	output, _ = runInTmux(t, home, socket, binary+" install", 120*time.Second)
	assertContains(t, output, `Already installed "tmux-sensible"`)
}
