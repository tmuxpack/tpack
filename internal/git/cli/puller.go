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
	pinned := false
	if opts.Branch != "" {
		// Best-effort: surface newly-published tags before checkout.
		fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--tags", "--force")
		fetchCmd.Dir = opts.Dir
		fetchCmd.Env = append(fetchCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		_ = fetchCmd.Run()

		// --end-of-options ensures a ref starting with "-" is treated as a
		// ref, not as a git option (the ref comes from untrusted config).
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "--end-of-options", opts.Branch)
		checkoutCmd.Dir = opts.Dir
		checkoutCmd.Env = append(checkoutCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		if err := checkoutCmd.Run(); err != nil {
			return "", fmt.Errorf("git checkout %s: %w", opts.Branch, err)
		}

		// Detached HEAD means a tag/SHA pin; skip pull so HEAD stays put.
		symCmd := exec.CommandContext(ctx, "git", "symbolic-ref", "-q", "HEAD")
		symCmd.Dir = opts.Dir
		symCmd.Env = append(symCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		pinned = symCmd.Run() != nil
	}

	var out []byte
	if pinned {
		out = []byte(fmt.Sprintf("pinned to %s", opts.Branch))
	} else {
		pullCmd := exec.CommandContext(ctx, "git", "pull", "--rebase=false")
		pullCmd.Dir = opts.Dir
		pullCmd.Env = append(pullCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		var err error
		out, err = pullCmd.CombinedOutput()
		if err != nil {
			return strings.TrimSpace(string(out)), err
		}
	}

	// Submodules are best-effort: a plugin whose submodule references an
	// unreachable commit (see issue #14) should still update. The failure
	// is surfaced via OnWarning so callers can notify the user.
	subCmd := exec.CommandContext(ctx, "git", "submodule", "update", "--init", "--recursive")
	subCmd.Dir = opts.Dir
	subCmd.Env = append(subCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if subOut, subErr := subCmd.CombinedOutput(); subErr != nil && opts.OnWarning != nil {
		opts.OnWarning("submodule update failed: " + strings.TrimSpace(string(subOut)))
	}

	return strings.TrimSpace(string(out)), nil
}
