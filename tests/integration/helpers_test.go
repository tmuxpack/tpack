package integration_test

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

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

func pluginPath(t *testing.T, rootPath string, p plug.Plugin) string {
	t.Helper()
	path, err := mustRoot(t, rootPath).Child(p.DirName)
	if err != nil {
		t.Fatalf("canonical path for %q: %v", p.Raw, err)
	}
	return path
}

func legacyPluginPath(t *testing.T, rootPath string, p plug.Plugin) string {
	t.Helper()
	path, err := mustRoot(t, rootPath).Child(plug.LegacyPluginName(p.Spec))
	if err != nil {
		t.Fatalf("legacy path for %q: %v", p.Raw, err)
	}
	return path
}

func createLocalRepository(t *testing.T, barePath, marker string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(barePath), 0o755); err != nil {
		t.Fatalf("create bare repository parent: %v", err)
	}
	runGit(t, "", "init", "--bare", "--initial-branch=main", barePath)

	workPath := t.TempDir()
	runGit(t, "", "init", "--initial-branch=main", workPath)
	runGit(t, workPath, "config", "user.name", "tpack integration")
	runGit(t, workPath, "config", "user.email", "integration@tpack.test")
	if err := os.WriteFile(filepath.Join(workPath, "marker"), []byte(marker), 0o644); err != nil {
		t.Fatalf("write repository marker: %v", err)
	}
	runGit(t, workPath, "add", "marker")
	runGit(t, workPath, "commit", "-m", "add marker")
	runGit(t, workPath, "remote", "add", "origin", barePath)
	runGit(t, workPath, "push", "-u", "origin", "main")
	return barePath
}

func configureGitURLRewrites(t *testing.T, rewrites map[string]string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	cloneURLs := make([]string, 0, len(rewrites))
	for cloneURL := range rewrites {
		cloneURLs = append(cloneURLs, cloneURL)
	}
	sort.Strings(cloneURLs)

	t.Setenv("GIT_CONFIG_COUNT", strconv.Itoa(len(cloneURLs)+1))
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
	for i, cloneURL := range cloneURLs {
		localURL := (&url.URL{Scheme: "file", Path: rewrites[cloneURL]}).String()
		t.Setenv("GIT_CONFIG_KEY_"+strconv.Itoa(i+1), "url."+localURL+".insteadOf")
		t.Setenv("GIT_CONFIG_VALUE_"+strconv.Itoa(i+1), cloneURL)
	}
}

func cloneRepository(t *testing.T, cloneURL, destination string) {
	t.Helper()
	runGit(t, "", "clone", cloneURL, destination)
}

func setRepositoryOrigin(t *testing.T, repository, origin string) {
	t.Helper()
	runGit(t, repository, "remote", "set-url", "origin", origin)
}

func assertMarker(t *testing.T, pluginDir, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(pluginDir, "marker"))
	if err != nil {
		t.Fatalf("read marker in %s: %v", pluginDir, err)
	}
	if string(got) != want {
		t.Errorf("marker in %s = %q, want %q", pluginDir, got, want)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
