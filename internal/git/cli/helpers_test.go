package cli_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

// Compile-time interface compliance checks.
var (
	_ git.Cloner    = (*gitcli.Cloner)(nil)
	_ git.Puller    = (*gitcli.Puller)(nil)
	_ git.Validator = (*gitcli.Validator)(nil)
	_ git.Fetcher   = (*gitcli.Fetcher)(nil)
	_ git.RevParser = (*gitcli.RevParser)(nil)
	_ git.Logger    = (*gitcli.Logger)(nil)
)

// initBareRepo creates a bare git repository with a single commit on the
// default branch. It returns the path to the bare repo directory.
func initBareRepo(t *testing.T) string {
	t.Helper()

	bare := filepath.Join(t.TempDir(), "bare.git")

	// Create the bare repo.
	runGit(t, "", "init", "--bare", bare)

	// Clone it into a temporary working copy so we can make the initial commit.
	work := filepath.Join(t.TempDir(), "work")
	runGit(t, "", "clone", bare, work)

	// Configure committer identity inside the working copy.
	runGit(t, work, "config", "user.email", "test@test.com")
	runGit(t, work, "config", "user.name", "Test")

	// Create an initial commit.
	writeFile(t, filepath.Join(work, "README"), "init")
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "initial commit")
	runGit(t, work, "push", "origin", "HEAD")

	return bare
}

// cloneLocal clones the bare repo into a new temp directory and returns its path.
func cloneLocal(t *testing.T, bareDir string) string {
	t.Helper()

	dst := filepath.Join(t.TempDir(), "clone")
	runGit(t, "", "clone", bareDir, dst)
	runGit(t, dst, "config", "user.email", "test@test.com")
	runGit(t, dst, "config", "user.name", "Test")
	return dst
}

// addCommitToBare clones the bare repo, adds a new file, commits, and pushes
// back to the bare repo so that it has a new commit that existing clones do
// not have.
func addCommitToBare(t *testing.T, bareDir, filename string) {
	t.Helper()

	work := filepath.Join(t.TempDir(), "pusher")
	runGit(t, "", "clone", bareDir, work)
	runGit(t, work, "config", "user.email", "test@test.com")
	runGit(t, work, "config", "user.name", "Test")

	writeFile(t, filepath.Join(work, filename), "content of "+filename)
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "add "+filename)
	runGit(t, work, "push", "origin", "HEAD")
}

// runGit executes a git command and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command git %v failed: %v\n%s", args, err, out)
	}
}

// writeFile writes content to a file, creating parent directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
