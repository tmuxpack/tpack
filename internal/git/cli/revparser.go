package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Resolves git refs using the git CLI.
type RevParser struct{}

func NewRevParser() *RevParser {
	return &RevParser{}
}

func (c *RevParser) RevParse(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}
