package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const originTimeout = 5 * time.Second

// OriginReader reads repository origins using the Git CLI.
type OriginReader struct{}

func NewOriginReader() *OriginReader {
	return &OriginReader{}
}

func (r *OriginReader) Origin(ctx context.Context, dir string) (string, error) {
	childCtx, cancel := context.WithTimeout(ctx, originTimeout)
	defer cancel()

	cmd := exec.CommandContext(childCtx, "git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read Git origin in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
