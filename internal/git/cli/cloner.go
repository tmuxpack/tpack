package cli

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/tmuxpack/tpack/internal/git"
)

// Cloner clones git repositories using the git CLI.
type Cloner struct{}

// NewCloner returns a new Cloner.
func NewCloner() *Cloner {
	return &Cloner{}
}

func (c *Cloner) Clone(ctx context.Context, opts git.CloneOptions) error {
	args := []string{"clone", "--single-branch", "--recursive"}
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
