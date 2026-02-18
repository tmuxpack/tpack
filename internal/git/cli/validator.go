package cli

import "os/exec"

// Checks if a directory is a git repo using the git CLI.
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (c *Validator) IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir") //nolint:noctx // fast local check, no cancellation needed
	return cmd.Run() == nil
}
