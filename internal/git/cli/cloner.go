package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tmuxpack/tpack/internal/git"
)

// Cloner clones git repositories using the git CLI.
type Cloner struct{}

func NewCloner() *Cloner {
	return &Cloner{}
}

func (c *Cloner) Clone(ctx context.Context, opts git.CloneOptions) error {
	// First try the ref as a branch or tag: `git clone -b` resolves both, so a
	// branch/tag whose name happens to look like a hex SHA (e.g. "deadbeef") is
	// handled here and never misclassified. git also keeps the requested depth.
	err := c.cloneBranch(ctx, opts)
	if err == nil {
		return c.initSubmodules(ctx, opts)
	}

	// A real commit SHA is rejected by `git clone -b`. Only for SHA-shaped refs
	// do we fall back to a full clone + checkout (a SHA may live anywhere in
	// history, so the shallow/single-branch optimization cannot apply).
	if !looksLikeCommitSHA(opts.Branch) {
		return err
	}
	if err := c.cloneSHA(ctx, opts); err != nil {
		return err
	}
	return c.initSubmodules(ctx, opts)
}

func (c *Cloner) cloneBranch(ctx context.Context, opts git.CloneOptions) error {
	args := []string{"clone", "--single-branch"}
	if opts.Depth > 0 {
		args = append(args, "--depth", strconv.Itoa(opts.Depth))
	}
	if opts.Branch != "" {
		args = append(args, "-b", opts.Branch)
	}
	args = append(args, opts.URL, opts.Dir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run()
}

func (c *Cloner) cloneSHA(ctx context.Context, opts git.CloneOptions) error {
	cloneCmd := exec.CommandContext(ctx, "git", "clone", opts.URL, opts.Dir)
	cloneCmd.Env = append(cloneCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cloneCmd.Run(); err != nil {
		return err
	}

	// --end-of-options keeps a ref starting with "-" from being parsed as a
	// git option (the ref comes from untrusted config).
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "--end-of-options", opts.Branch)
	checkoutCmd.Dir = opts.Dir
	checkoutCmd.Env = append(checkoutCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %w: %s", opts.Branch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Initializes submodules best-effort: a plugin whose submodule
// references an unreachable commit (see TPM issue #14) should still install.
func (c *Cloner) initSubmodules(ctx context.Context, opts git.CloneOptions) error {
	subCmd := exec.CommandContext(ctx, "git", "submodule", "update", "--init", "--recursive")
	subCmd.Dir = opts.Dir
	subCmd.Env = append(subCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if subOut, subErr := subCmd.CombinedOutput(); subErr != nil && opts.OnWarning != nil {
		opts.OnWarning("submodule update failed: " + strings.TrimSpace(string(subOut)))
	}
	return nil
}
