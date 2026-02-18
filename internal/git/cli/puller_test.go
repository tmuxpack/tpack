package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestPuller_PullUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	puller := gitcli.NewPuller()
	out, err := puller.Pull(context.Background(), git.PullOptions{Dir: clone})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	if out == "" {
		t.Fatal("expected non-empty output from pull")
	}
}

func TestPuller_PullWithUpstreamChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Push a new commit to the bare repo after the clone was made.
	addCommitToBare(t, bare, "new-file.txt")

	puller := gitcli.NewPuller()
	out, err := puller.Pull(context.Background(), git.PullOptions{Dir: clone})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	if out == "" {
		t.Fatal("expected non-empty output from pull with changes")
	}

	// The new file should now be present.
	if _, err := os.Stat(filepath.Join(clone, "new-file.txt")); err != nil {
		t.Fatalf("expected new-file.txt after pull: %v", err)
	}
}

func TestPuller_PullNonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dir := t.TempDir() // plain directory, not a git repo

	puller := gitcli.NewPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{Dir: dir})
	if err == nil {
		t.Fatal("expected error when pulling in non-git directory")
	}
}

func TestPuller_PullWithBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)

	// Create a feature branch.
	work := cloneLocal(t, bare)
	runGit(t, work, "checkout", "-b", "feature")
	writeFile(t, filepath.Join(work, "feature.txt"), "on feature")
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "feature commit")
	runGit(t, work, "push", "origin", "feature")

	// Clone (will be on default branch).
	clone := cloneLocal(t, bare)

	// Pull with branch should checkout feature and pull.
	puller := gitcli.NewPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{
		Dir:    clone,
		Branch: "feature",
	})
	if err != nil {
		t.Fatalf("Pull with branch returned error: %v", err)
	}

	// feature.txt should now exist.
	if _, err := os.Stat(filepath.Join(clone, "feature.txt")); err != nil {
		t.Fatalf("expected feature.txt after pull with branch: %v", err)
	}
}
