package cli_test

import (
	"context"
	"testing"
	"time"

	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestFetcher_IsOutdatedUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	fetcher := gitcli.NewFetcher()
	outdated, err := fetcher.IsOutdated(context.Background(), clone)
	if err != nil {
		t.Fatalf("IsOutdated returned error: %v", err)
	}
	if outdated {
		t.Fatal("expected repo to be up-to-date")
	}
}

func TestFetcher_IsOutdatedWithNewCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	// Add a commit to the bare repo so the clone is behind.
	addCommitToBare(t, bare, "upstream-change.txt")

	fetcher := gitcli.NewFetcher()
	outdated, err := fetcher.IsOutdated(context.Background(), clone)
	if err != nil {
		t.Fatalf("IsOutdated returned error: %v", err)
	}
	if !outdated {
		t.Fatal("expected repo to be outdated after upstream commit")
	}
}

func TestFetcher_IsOutdatedNonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	dir := t.TempDir()

	fetcher := gitcli.NewFetcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := fetcher.IsOutdated(ctx, dir)
	if err == nil {
		t.Fatal("expected error when checking non-git directory")
	}
}
