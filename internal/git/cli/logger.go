package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tmuxpack/tpack/internal/git"
)

// Retrieves commit logs using the git CLI.
type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (c *Logger) Log(ctx context.Context, dir, fromRef, toRef string) ([]git.Commit, error) {
	cmd := exec.CommandContext(ctx, "git", "log", fromRef+".."+toRef, "--oneline", "--no-decorate")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log %s..%s in %s: %w", fromRef, toRef, dir, err)
	}

	var commits []git.Commit
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		hash, message, _ := strings.Cut(line, " ")
		commits = append(commits, git.Commit{Hash: hash, Message: message})
	}
	return commits, nil
}
