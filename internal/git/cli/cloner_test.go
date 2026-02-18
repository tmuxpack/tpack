package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestCloner_CloneSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	dst := filepath.Join(t.TempDir(), "cloned")

	cloner := gitcli.NewCloner()
	err := cloner.Clone(context.Background(), git.CloneOptions{
		URL: bare,
		Dir: dst,
	})
	if err != nil {
		t.Fatalf("Clone returned error: %v", err)
	}

	// The cloned directory must exist and contain the file from the initial commit.
	if _, err := os.Stat(filepath.Join(dst, "README")); err != nil {
		t.Fatalf("expected README in cloned repo: %v", err)
	}
}

func TestCloner_CloneWithBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)

	// Create a feature branch in the bare repo via a temporary working copy.
	work := cloneLocal(t, bare)
	runGit(t, work, "checkout", "-b", "feature")
	writeFile(t, filepath.Join(work, "feature.txt"), "on feature branch")
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "feature commit")
	runGit(t, work, "push", "origin", "feature")

	// Clone specifying the feature branch.
	dst := filepath.Join(t.TempDir(), "cloned-branch")
	cloner := gitcli.NewCloner()
	err := cloner.Clone(context.Background(), git.CloneOptions{
		URL:    bare,
		Dir:    dst,
		Branch: "feature",
	})
	if err != nil {
		t.Fatalf("Clone with branch returned error: %v", err)
	}

	// feature.txt should be present because we cloned the feature branch.
	if _, err := os.Stat(filepath.Join(dst, "feature.txt")); err != nil {
		t.Fatalf("expected feature.txt in cloned repo: %v", err)
	}
}

func TestCloner_CloneInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dst := filepath.Join(t.TempDir(), "bad-clone")
	cloner := gitcli.NewCloner()
	err := cloner.Clone(context.Background(), git.CloneOptions{
		URL: "/nonexistent/path/to/repo.git",
		Dir: dst,
	})
	if err == nil {
		t.Fatal("expected error when cloning invalid URL")
	}
}

func TestCloner_CloneCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	dst := filepath.Join(t.TempDir(), "canceled-clone")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cloner := gitcli.NewCloner()
	err := cloner.Clone(ctx, git.CloneOptions{
		URL: bare,
		Dir: dst,
	})
	if err == nil {
		t.Fatal("expected error when context is canceled")
	}
}
