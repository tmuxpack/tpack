package cli

import (
	"context"
	"os/exec"
	"time"
)

const gitRepoCheckTimeout = 5 * time.Second

// Checks if a directory is a git repo using the git CLI.
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (c *Validator) IsGitRepo(dir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), gitRepoCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--git-dir") //nolint:gosec // dir is from resolved plugin path
	return cmd.Run() == nil
}
