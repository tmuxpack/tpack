package git

import (
	"context"
	"os/exec"
	"strings"
)

// CLICloner clones git repositories using the git CLI.
type CLICloner struct{}

// NewCLICloner returns a new CLICloner.
func NewCLICloner() *CLICloner {
	return &CLICloner{}
}

func (c *CLICloner) Clone(ctx context.Context, opts CloneOptions) error {
	args := []string{"clone", "--single-branch", "--recursive"}
	if opts.Branch != "" {
		args = append(args, "-b", opts.Branch)
	}
	args = append(args, opts.URL, opts.Dir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run()
}

// CLIPuller pulls updates using the git CLI.
type CLIPuller struct{}

// NewCLIPuller returns a new CLIPuller.
func NewCLIPuller() *CLIPuller {
	return &CLIPuller{}
}

func (c *CLIPuller) Pull(ctx context.Context, opts PullOptions) (string, error) {
	// git pull
	pullCmd := exec.CommandContext(ctx, "git", "pull")
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

// CLIValidator checks if a directory is a git repo using the git CLI.
type CLIValidator struct{}

// NewCLIValidator returns a new CLIValidator.
func NewCLIValidator() *CLIValidator {
	return &CLIValidator{}
}

func (c *CLIValidator) IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "remote") //nolint:noctx // fast local check, no cancellation needed
	cmd.Dir = dir
	return cmd.Run() == nil
}
