package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Checks outdated status by fetching and comparing refs via the git CLI.
type Fetcher struct{}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

func (c *Fetcher) IsOutdated(ctx context.Context, dir string) (bool, error) {
	// symbolic-ref -q HEAD: exit 0 = on a branch, exit 1 = detached HEAD
	// (pinned to a tag/SHA; never outdated). Any other failure (not a repo,
	// git missing, ...) is a real error and must be surfaced, not swallowed.
	symCmd := exec.CommandContext(ctx, "git", "symbolic-ref", "-q", "HEAD")
	symCmd.Dir = dir
	if err := symCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git symbolic-ref in %s: %w", dir, err)
	}

	fetchCmd := exec.CommandContext(ctx, "git", "fetch")
	fetchCmd.Dir = dir
	fetchCmd.Env = append(fetchCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := fetchCmd.Run(); err != nil {
		return false, fmt.Errorf("git fetch in %s: %w", dir, err)
	}

	localCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	localCmd.Dir = dir
	localOut, err := localCmd.Output()
	if err != nil {
		return false, fmt.Errorf("rev-parse HEAD in %s: %w", dir, err)
	}

	remoteCmd := exec.CommandContext(ctx, "git", "rev-parse", "@{u}")
	remoteCmd.Dir = dir
	remoteOut, err := remoteCmd.Output()
	if err != nil {
		return false, fmt.Errorf("rev-parse @{u} in %s: %w", dir, err)
	}

	return strings.TrimSpace(string(localOut)) != strings.TrimSpace(string(remoteOut)), nil
}
