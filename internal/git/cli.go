package git

import (
	"context"
	"fmt"
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

// CLIFetcher checks outdated status by fetching and comparing refs via the git CLI.
type CLIFetcher struct{}

// NewCLIFetcher returns a new CLIFetcher.
func NewCLIFetcher() *CLIFetcher {
	return &CLIFetcher{}
}

// CLIRevParser resolves git refs using the git CLI.
type CLIRevParser struct{}

// NewCLIRevParser returns a new CLIRevParser.
func NewCLIRevParser() *CLIRevParser {
	return &CLIRevParser{}
}

func (c *CLIRevParser) RevParse(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CLILogger retrieves commit logs using the git CLI.
type CLILogger struct{}

// NewCLILogger returns a new CLILogger.
func NewCLILogger() *CLILogger {
	return &CLILogger{}
}

func (c *CLILogger) Log(ctx context.Context, dir, fromRef, toRef string) ([]Commit, error) {
	cmd := exec.CommandContext(ctx, "git", "log", fromRef+".."+toRef, "--oneline", "--no-decorate")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log %s..%s in %s: %w", fromRef, toRef, dir, err)
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		hash, message, _ := strings.Cut(line, " ")
		commits = append(commits, Commit{Hash: hash, Message: message})
	}
	return commits, nil
}

func (c *CLIFetcher) IsOutdated(ctx context.Context, dir string) (bool, error) {
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
