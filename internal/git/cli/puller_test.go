package cli_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// revParse returns the resolved SHA of HEAD (or any ref) for a repo.
func revParse(t *testing.T, dir, ref string) string {
	t.Helper()
	out, err := exec.CommandContext(context.Background(),
		"git", "-C", dir, "rev-parse", ref).Output()
	if err != nil {
		t.Fatalf("rev-parse %s in %s: %v", ref, dir, err)
	}
	return strings.TrimSpace(string(out))
}

// A ref starting with "-" must be treated as a ref (and fail to resolve),
// never interpreted as a git option. Branch values come from untrusted config.
func TestPuller_PullRejectsOptionLikeRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)
	before := revParse(t, clone, "HEAD")

	puller := gitcli.NewPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{
		Dir:    clone,
		Branch: "-q", // swallowed as the `git checkout -q` flag without --end-of-options
	})
	if err == nil {
		t.Fatal("expected error for option-like ref; it was likely interpreted as a git flag")
	}

	// HEAD must be untouched by the failed checkout.
	if got := revParse(t, clone, "HEAD"); got != before {
		t.Fatalf("HEAD moved after failed checkout: HEAD=%s, want %s", got, before)
	}
}

func TestPuller_PullSkippedOnTag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)

	// Tag the initial commit and push the tag upstream.
	tagger := cloneLocal(t, bare)
	runGit(t, tagger, "tag", "v1.0.0")
	runGit(t, tagger, "push", "origin", "v1.0.0")
	tagSHA := revParse(t, tagger, "v1.0.0")

	// Clone fresh, then check out the tag so HEAD is detached.
	clone := cloneLocal(t, bare)
	runGit(t, clone, "fetch", "--tags")
	runGit(t, clone, "checkout", "v1.0.0")

	// Push a new commit upstream so a naive `git pull` would fast-forward.
	addCommitToBare(t, bare, "after-tag.txt")

	puller := gitcli.NewPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{
		Dir:    clone,
		Branch: "v1.0.0",
	})
	if err != nil {
		t.Fatalf("Pull pinned to tag returned error: %v", err)
	}

	if got := revParse(t, clone, "HEAD"); got != tagSHA {
		t.Fatalf("HEAD moved off tag: HEAD=%s, want %s", got, tagSHA)
	}

	// The new file must NOT be present; pull should have been skipped.
	if _, err := os.Stat(filepath.Join(clone, "after-tag.txt")); err == nil {
		t.Fatal("after-tag.txt should not exist; pull should have been skipped on detached HEAD")
	}
}

func TestPuller_PullSkippedOnCommitSHA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)

	// Capture the initial commit SHA and check it out (detached HEAD).
	clone := cloneLocal(t, bare)
	sha := revParse(t, clone, "HEAD")
	runGit(t, clone, "checkout", sha)

	// Push a new commit upstream so a naive `git pull` would fast-forward.
	addCommitToBare(t, bare, "after-sha.txt")

	puller := gitcli.NewPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{
		Dir:    clone,
		Branch: sha,
	})
	if err != nil {
		t.Fatalf("Pull pinned to SHA returned error: %v", err)
	}

	if got := revParse(t, clone, "HEAD"); got != sha {
		t.Fatalf("HEAD moved off pinned SHA: HEAD=%s, want %s", got, sha)
	}

	if _, err := os.Stat(filepath.Join(clone, "after-sha.txt")); err == nil {
		t.Fatal("after-sha.txt should not exist; pull should have been skipped on detached HEAD")
	}
}
