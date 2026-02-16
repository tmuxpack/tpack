package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	buildOnce    sync.Once
	builtBinPath string
	buildErr     error
)

// projectRoot returns the absolute path to the repository root.
// Since this file lives at tests/e2e/, we go two levels up.
func projectRoot() string {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		panic(fmt.Sprintf("failed to resolve project root: %v", err))
	}
	return root
}

// buildBinary compiles the tpm binary once per test invocation using sync.Once.
// Returns the path to the compiled binary. Skips the test on build failure.
func buildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, dirErr := os.MkdirTemp("", "tpm-e2e-bin-*")
		if dirErr != nil {
			buildErr = dirErr
			return
		}
		builtBinPath = filepath.Join(dir, "tpm")
		cmd := exec.CommandContext(context.Background(), "go", "build", "-o", builtBinPath, "./cmd/tpm")
		cmd.Dir = projectRoot()
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("build failed: %w\n%s", err, out)
		}
	})

	if buildErr != nil {
		t.Skipf("failed to build tpm binary: %v", buildErr)
	}

	return builtBinPath
}

// socketName returns a unique tmux socket name for test isolation.
// Format: tpm-e2e-<sanitized name>-<nanotime mod>.
// Kept under 40 characters to avoid socket path length limits.
func socketName(t *testing.T) string {
	t.Helper()

	name := t.Name()
	// Replace non-alphanumeric characters with dashes.
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)

	nanoMod := time.Now().UnixNano() % 1_000_000

	sock := fmt.Sprintf("tpm-e2e-%s-%d", sanitized, nanoMod)
	if len(sock) > 40 {
		sock = sock[:40]
	}

	return sock
}

// e2eEnv creates an isolated HOME directory for a test.
// It writes tmuxConf to $HOME/.tmux.conf and creates $HOME/.tmux/plugins/.
// Returns the home directory path and a unique socket name.
func e2eEnv(t *testing.T, tmuxConf string) (home, socket string) {
	t.Helper()

	home = t.TempDir()
	socket = socketName(t)

	confPath := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(confPath, []byte(tmuxConf), 0o644); err != nil {
		t.Fatalf("failed to write tmux.conf: %v", err)
	}

	pluginDir := filepath.Join(home, ".tmux", "plugins")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin directory: %v", err)
	}

	return home, socket
}

// cleanEnv builds an environment slice from os.Environ() with HOME set to the
// given value and TMUX/TMUX_PANE removed. This avoids two problems:
//   - Duplicate HOME entries (C getenv returns the first match, Go's os.Getenv
//     may return a different one, leading to subtle bugs).
//   - Leaking the parent TMUX socket into child tmux servers, which causes
//     commands inside run-shell to target the wrong server.
func cleanEnv(home string) []string {
	var env []string
	for _, e := range os.Environ() {
		key := e[:strings.Index(e, "=")+1]
		switch key {
		case "HOME=", "TMUX=", "TMUX_PANE=", "TMUX_PLUGIN_MANAGER_PATH=":
			continue
		}
		env = append(env, e)
	}
	return append(env, "HOME="+home)
}

// startTmux starts a tmux server with the given socket and HOME.
// It registers a cleanup function to kill the server when the test finishes.
// Polls tmux list-sessions until the server is ready (5s timeout).
func startTmux(t *testing.T, home, socket string) {
	t.Helper()

	confPath := filepath.Join(home, ".tmux.conf")
	env := cleanEnv(home)

	cmd := exec.CommandContext(context.Background(), "tmux", "-L", socket, "-f", confPath, "new-session", "-d")
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to start tmux: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		kill := exec.CommandContext(context.Background(), "tmux", "-L", socket, "kill-server")
		kill.Env = env
		_ = kill.Run()
	})

	// Poll until tmux is ready.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		check := exec.CommandContext(context.Background(), "tmux", "-L", socket, "list-sessions")
		check.Env = env
		if err := check.Run(); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("tmux server did not become ready within 5 seconds")
}

// runInTmux runs a command inside the tmux session via tmux run-shell.
// The command's stdout+stderr are captured to a temp file and an exit code
// is written to a separate file. Polls for the exit file until timeout.
// Returns (combined output, exit code).
func runInTmux(t *testing.T, home, socket, command string, timeout time.Duration) (string, int) {
	t.Helper()

	outFile := filepath.Join(t.TempDir(), "output.txt")
	exitFile := filepath.Join(t.TempDir(), "exit.txt")

	wrapped := fmt.Sprintf(
		"HOME=%s %s > %s 2>&1; echo $? > %s",
		home, command, outFile, exitFile,
	)

	cmd := exec.CommandContext(context.Background(), "tmux", "-L", socket, "run-shell", wrapped)
	cmd.Env = append(os.Environ(), "HOME="+home)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tmux run-shell failed: %v\n%s", err, out)
	}

	// Poll for the exit file.
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(exitFile); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if _, err := os.Stat(exitFile); err != nil {
		t.Fatalf("exit file not created within %v", timeout)
	}

	outputBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	exitBytes, err := os.ReadFile(exitFile)
	if err != nil {
		t.Fatalf("failed to read exit file: %v", err)
	}

	exitCode, err := strconv.Atoi(strings.TrimSpace(string(exitBytes)))
	if err != nil {
		t.Fatalf("failed to parse exit code %q: %v", string(exitBytes), err)
	}

	return string(outputBytes), exitCode
}

// tmuxShowEnv reads a tmux environment variable via show-environment -g.
// Returns the value, or fails the test if the variable is not found.
func tmuxShowEnv(t *testing.T, socket, name string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "tmux", "-L", socket, "show-environment", "-g", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tmux show-environment failed for %q: %v\n%s", name, err, out)
	}

	line := strings.TrimSpace(string(out))
	// Format is NAME=value
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected show-environment output for %q: %q", name, line)
	}

	return parts[1]
}

// tmuxListKeys returns the output of tmux list-keys for the given socket.
func tmuxListKeys(t *testing.T, socket string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "tmux", "-L", socket, "list-keys")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tmux list-keys failed: %v\n%s", err, out)
	}

	return string(out)
}

// installPluginManually clones a GitHub repository into the plugin directory.
// The repo should be in "owner/name" format (e.g., "tmux-plugins/tmux-example-plugin").
func installPluginManually(t *testing.T, pluginDir, repo string) {
	t.Helper()

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid repo format %q, expected owner/name", repo)
	}
	name := parts[1]

	destDir := filepath.Join(pluginDir, name)
	url := "https://github.com/" + repo

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, destDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to clone %s: %v\n%s", repo, err, out)
	}
}

// assertContains checks that output contains the expected substring.
func assertContains(t *testing.T, output, expected string) {
	t.Helper()

	if !strings.Contains(output, expected) {
		t.Errorf("expected output to contain %q, got:\n%s", expected, output)
	}
}

// assertDirExists checks that the given path exists and is a directory.
func assertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected directory %q to exist, got error: %v", path, err)
		return
	}

	if !info.IsDir() {
		t.Errorf("expected %q to be a directory, but it is not", path)
	}
}

// assertDirNotExists checks that the given path does not exist.
func assertDirNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Errorf("expected directory %q to not exist, but it does", path)
		return
	}

	if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking %q: %v", path, err)
	}
}

// skipIfNoTmux skips the test if tmux is not available on the system.
func skipIfNoTmux(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found, skipping E2E test")
	}
}

// skipIfNoGit skips the test if git is not available on the system.
func skipIfNoGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping E2E test")
	}
}

// waitForFile polls for a file to exist, checking every 100ms up to the given timeout.
func waitForFile(t *testing.T, path string, timeoutSec int) {
	t.Helper()

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("file %q did not appear within %d seconds", path, timeoutSec)
}

// waitForEnv polls a tmux environment variable until it matches the expected value.
// Checks every 100ms up to the given timeout.
//
//nolint:unparam // name is always TMUX_PLUGIN_MANAGER_PATH today but kept generic for future tests.
func waitForEnv(t *testing.T, socket, name, expected string, timeoutSec int) {
	t.Helper()

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	var lastVal string

	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(context.Background(), "tmux", "-L", socket, "show-environment", "-g", name)
		out, err := cmd.CombinedOutput()
		if err == nil {
			line := strings.TrimSpace(string(out))
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				lastVal = parts[1]
				if lastVal == expected {
					return
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("tmux env %q did not become %q within %d seconds (last value: %q)",
		name, expected, timeoutSec, lastVal)
}
