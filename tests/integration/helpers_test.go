package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func mustParsePlugin(t *testing.T, raw string) plug.Plugin {
	t.Helper()
	p, err := plug.ParseSpec(raw, nil)
	if err != nil {
		t.Fatalf("ParseSpec(%q): %v", raw, err)
	}
	return p
}

const tmuxExamplePlugin = "tmux-plugins/tmux-example-plugin"

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping integration test")
	}
}

func mustRoot(t *testing.T, path string) plug.Root {
	t.Helper()
	root, err := plug.NewRoot("test", path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return root
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
