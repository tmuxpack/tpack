package cli_test

import (
	"context"
	"strings"
	"testing"

	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestOriginReaderReadsOrigin(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "remote", "add", "origin", "git@github.com:catppuccin/tmux.git")

	got, err := gitcli.NewOriginReader().Origin(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "git@github.com:catppuccin/tmux.git" {
		t.Fatalf("origin = %q", got)
	}
}

func TestOriginReaderReportsDirectoryForNonRepository(t *testing.T) {
	dir := t.TempDir()

	_, err := gitcli.NewOriginReader().Origin(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), dir) || !strings.Contains(err.Error(), "git") {
		t.Fatalf("error = %q, want directory and Git context", err)
	}
}

func TestOriginReaderReportsDirectoryForMissingOrigin(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")

	_, err := gitcli.NewOriginReader().Origin(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), dir) || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("error = %q, want directory and origin context", err)
	}
}
