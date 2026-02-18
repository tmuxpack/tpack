package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tmuxpack/tpack/internal/git"
)

// Pulls updates for an existing repository using the git CLI.
type Puller struct{}

func NewPuller() *Puller {
	return &Puller{}
}

func (c *Puller) Pull(ctx context.Context, opts git.PullOptions) (string, error) {
	if opts.Branch != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", opts.Branch)
		checkoutCmd.Dir = opts.Dir
		checkoutCmd.Env = append(checkoutCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		if err := checkoutCmd.Run(); err != nil {
			return "", fmt.Errorf("git checkout %s: %w", opts.Branch, err)
		}
	}

	// git pull
	pullCmd := exec.CommandContext(ctx, "git", "pull", "--rebase=false")
	pullCmd.Dir = opts.Dir
	pullCmd.Env = append(pullCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := pullCmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}

	// git submodule update --init --recursive
	subCmd := exec.CommandContext(ctx, "git", "submodule", "update", "--init", "--recursive")
	subCmd.Dir = opts.Dir
	subCmd.Env = append(subCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	subOut, subErr := subCmd.CombinedOutput()

	combined := strings.TrimSpace(string(out) + string(subOut))
	return combined, subErr
}
