package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/tui"
)

func runCommits(args []string) int {
	dir := flagValue(args, "--dir")
	from := flagValue(args, "--from")
	to := flagValue(args, "--to")
	name := flagValue(args, "--name")

	if dir == "" || from == "" || to == "" || name == "" {
		fmt.Fprintln(os.Stderr, "tpm commits: --dir, --from, --to, and --name are required")
		return 1
	}

	// Run git log to get commits.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := git.NewCLILogger()
	commits, err := logger.Log(ctx, dir, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm commits: git log failed:", err)
		return 1
	}

	if len(commits) == 0 {
		return 0
	}

	if err := tui.RunCommitViewer(name, commits); err != nil {
		fmt.Fprintln(os.Stderr, "tpm:", err)
		return 1
	}
	return 0
}

// flagValue returns the value after the given flag in args, or empty string.
func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, flag+"=") {
			return a[len(flag)+1:]
		}
	}
	return ""
}
