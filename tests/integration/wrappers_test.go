package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWrapperUsesRepositoryQualifiedExamples(t *testing.T) {
	repoRoot := integrationProjectRoot(t)
	fixtureRoot := t.TempDir()
	for _, relativePath := range []string{
		filepath.Join("bin", "update_plugins"),
		filepath.Join("lib", "find_binary.sh"),
		filepath.Join("lib", "download_binary.sh"),
	} {
		contents, err := os.ReadFile(filepath.Join(repoRoot, relativePath))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(fixtureRoot, relativePath)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, contents, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}
	dirname, err := exec.LookPath("dirname")
	if err != nil {
		t.Skip("dirname not found")
	}
	toolDir := t.TempDir()
	if err = os.Symlink(dirname, filepath.Join(toolDir, "dirname")); err != nil {
		t.Skipf("cannot create dirname symlink: %v", err)
	}
	writeExecutable(t, filepath.Join(toolDir, "tpack"))

	cmd := exec.CommandContext(context.Background(), bash, filepath.Join(fixtureRoot, "bin", "update_plugins"))
	cmd.Env = append(envWithPath(toolDir), "TPACK_AUTO_DOWNLOAD=0")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("wrapper without arguments unexpectedly succeeded")
	}
	if !strings.Contains(string(output), "tmux-plugins/tmux-sensible") {
		t.Fatalf("wrapper usage lacks a repository-qualified example:\n%s", output)
	}
	if strings.Contains(string(output), "update plugin 'tmux-foo'") {
		t.Fatalf("wrapper still advertises basename-only updates:\n%s", output)
	}
}
