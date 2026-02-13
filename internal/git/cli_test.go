package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/git"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// CLICloner
// ---------------------------------------------------------------------------

func TestCLICloner_CloneSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	dst := filepath.Join(t.TempDir(), "cloned")

	cloner := git.NewCLICloner()
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

func TestCLICloner_CloneWithBranch(t *testing.T) {
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
	cloner := git.NewCLICloner()
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

func TestCLICloner_CloneInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dst := filepath.Join(t.TempDir(), "bad-clone")
	cloner := git.NewCLICloner()
	err := cloner.Clone(context.Background(), git.CloneOptions{
		URL: "/nonexistent/path/to/repo.git",
		Dir: dst,
	})
	if err == nil {
		t.Fatal("expected error when cloning invalid URL")
	}
}

func TestCLICloner_CloneCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	dst := filepath.Join(t.TempDir(), "canceled-clone")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cloner := git.NewCLICloner()
	err := cloner.Clone(ctx, git.CloneOptions{
		URL: bare,
		Dir: dst,
	})
	if err == nil {
		t.Fatal("expected error when context is canceled")
	}
}

// ---------------------------------------------------------------------------
// CLIPuller
// ---------------------------------------------------------------------------

func TestCLIPuller_PullUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	puller := git.NewCLIPuller()
	out, err := puller.Pull(context.Background(), git.PullOptions{Dir: clone})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}

	if out == "" {
		t.Fatal("expected non-empty output from pull")
	}
}

func TestCLIPuller_PullWithUpstreamChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Push a new commit to the bare repo after the clone was made.
	addCommitToBare(t, bare, "new-file.txt")

	puller := git.NewCLIPuller()
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

func TestCLIPuller_PullNonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dir := t.TempDir() // plain directory, not a git repo

	puller := git.NewCLIPuller()
	_, err := puller.Pull(context.Background(), git.PullOptions{Dir: dir})
	if err == nil {
		t.Fatal("expected error when pulling in non-git directory")
	}
}

func TestCLIPuller_PullWithBranch(t *testing.T) {
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
	puller := git.NewCLIPuller()
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

// ---------------------------------------------------------------------------
// CLIRevParser
// ---------------------------------------------------------------------------

func TestCLIRevParser_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	rp := git.NewCLIRevParser()
	hash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse returned error: %v", err)
	}
	if len(hash) < 7 {
		t.Errorf("expected commit hash, got %q", hash)
	}
}

func TestCLIRevParser_NonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	rp := git.NewCLIRevParser()
	_, err := rp.RevParse(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

// ---------------------------------------------------------------------------
// CLILogger
// ---------------------------------------------------------------------------

func TestCLILogger_LogWithCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Record current HEAD.
	rp := git.NewCLIRevParser()
	beforeHash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse before: %v", err)
	}

	// Push 2 new commits to the bare repo.
	addCommitToBare(t, bare, "file1.txt")
	addCommitToBare(t, bare, "file2.txt")

	// Pull to get the new commits.
	puller := git.NewCLIPuller()
	_, err = puller.Pull(context.Background(), git.PullOptions{Dir: clone})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}

	afterHash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse after: %v", err)
	}

	logger := git.NewCLILogger()
	commits, err := logger.Log(context.Background(), clone, beforeHash, afterHash)
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	// Commits should be in reverse chronological order.
	if commits[0].Message != "add file2.txt" {
		t.Errorf("expected first commit message 'add file2.txt', got %q", commits[0].Message)
	}
	if commits[1].Message != "add file1.txt" {
		t.Errorf("expected second commit message 'add file1.txt', got %q", commits[1].Message)
	}
}

func TestCLILogger_LogNoCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	rp := git.NewCLIRevParser()
	hash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse: %v", err)
	}

	logger := git.NewCLILogger()
	commits, err := logger.Log(context.Background(), clone, hash, hash)
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for same ref, got %d", len(commits))
	}
}

// ---------------------------------------------------------------------------
// CLIValidator
// ---------------------------------------------------------------------------

func TestCLIValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{
			name: "valid git repo",
			dir:  clone,
			want: true,
		},
		{
			name: "non-git directory",
			dir:  t.TempDir(),
			want: false,
		},
		{
			name: "nonexistent directory",
			dir:  filepath.Join(t.TempDir(), "does-not-exist"),
			want: false,
		},
	}

	v := git.NewCLIValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.IsGitRepo(tt.dir)
			if got != tt.want {
				t.Errorf("IsGitRepo(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CLIFetcher
// ---------------------------------------------------------------------------

func TestCLIFetcher_IsOutdatedUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	fetcher := git.NewCLIFetcher()
	outdated, err := fetcher.IsOutdated(context.Background(), clone)
	if err != nil {
		t.Fatalf("IsOutdated returned error: %v", err)
	}
	if outdated {
		t.Fatal("expected repo to be up-to-date")
	}
}

func TestCLIFetcher_IsOutdatedWithNewCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Add a commit to the bare repo so the clone is behind.
	addCommitToBare(t, bare, "upstream-change.txt")

	fetcher := git.NewCLIFetcher()
	outdated, err := fetcher.IsOutdated(context.Background(), clone)
	if err != nil {
		t.Fatalf("IsOutdated returned error: %v", err)
	}
	if !outdated {
		t.Fatal("expected repo to be outdated after upstream commit")
	}
}

func TestCLIFetcher_IsOutdatedNonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dir := t.TempDir()

	fetcher := git.NewCLIFetcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := fetcher.IsOutdated(ctx, dir)
	if err == nil {
		t.Fatal("expected error when checking non-git directory")
	}
}
