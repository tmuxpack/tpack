package cli_test

import (
	"context"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestLogger_LogWithCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Record current HEAD.
	rp := gitcli.NewRevParser()
	beforeHash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse before: %v", err)
	}

	// Push 2 new commits to the bare repo.
	addCommitToBare(t, bare, "file1.txt")
	addCommitToBare(t, bare, "file2.txt")

	// Pull to get the new commits.
	puller := gitcli.NewPuller()
	_, err = puller.Pull(context.Background(), git.PullOptions{Dir: clone})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}

	afterHash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse after: %v", err)
	}

	logger := gitcli.NewLogger()
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

func TestLogger_LogNoCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	rp := gitcli.NewRevParser()
	hash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse: %v", err)
	}

	logger := gitcli.NewLogger()
	commits, err := logger.Log(context.Background(), clone, hash, hash)
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for same ref, got %d", len(commits))
	}
}
