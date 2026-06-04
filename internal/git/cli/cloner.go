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

// NewCloner returns a new Cloner.
func NewCloner() *Cloner {
	return &Cloner{}
}

func (c *Cloner) Clone(ctx context.Context, opts git.CloneOptions) error {
	isSHA := opts.Branch != "" && looksLikeCommitSHA(opts.Branch)

	args := []string{"clone"}
	if !isSHA {
		args = append(args, "--single-branch")
		if opts.Depth > 0 {
			args = append(args, "--depth", strconv.Itoa(opts.Depth))
		}
		if opts.Branch != "" {
			args = append(args, "-b", opts.Branch)
		}
	}
	args = append(args, opts.URL, opts.Dir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err != nil {
		return err
	}

	if isSHA {
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", opts.Branch)
		checkoutCmd.Dir = opts.Dir
		checkoutCmd.Env = append(checkoutCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := checkoutCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git checkout %s: %w: %s", opts.Branch, err, strings.TrimSpace(string(out)))
		}
	}

	// Submodules are best-effort: a plugin whose submodule references an
	// unreachable commit (see issue #14) should still install. The failure
	// is surfaced via OnWarning so callers can notify the user.
	subCmd := exec.CommandContext(ctx, "git", "submodule", "update", "--init", "--recursive")
	subCmd.Dir = opts.Dir
	subCmd.Env = append(subCmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if subOut, subErr := subCmd.CombinedOutput(); subErr != nil && opts.OnWarning != nil {
		opts.OnWarning("submodule update failed: " + strings.TrimSpace(string(subOut)))
	}
	return nil
}
