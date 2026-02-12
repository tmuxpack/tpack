package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const tmuxExamplePlugin = "tmux-plugins/tmux-example-plugin"

func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found, skipping integration test")
	}
}

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping integration test")
	}
}

func setupIntegrationDir(t *testing.T) (pluginDir, confFile string) {
	t.Helper()
	dir := t.TempDir()
	pluginDir = filepath.Join(dir, "plugins") + "/"
	os.MkdirAll(pluginDir, 0o755)
	confFile = filepath.Join(dir, "tmux.conf")
	return pluginDir, confFile
}

func writeConf(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tpm-go")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/tpm")
	cmd.Dir = findProjectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return bin
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test directory to find go.mod.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
