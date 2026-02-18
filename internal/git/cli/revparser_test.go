package cli_test

import (
	"context"
	"testing"

	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestRevParser_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	rp := gitcli.NewRevParser()
	hash, err := rp.RevParse(context.Background(), clone)
	if err != nil {
		t.Fatalf("RevParse returned error: %v", err)
	}
	if len(hash) < 7 {
		t.Errorf("expected commit hash, got %q", hash)
	}
}

func TestRevParser_NonGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	rp := gitcli.NewRevParser()
	_, err := rp.RevParse(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}
