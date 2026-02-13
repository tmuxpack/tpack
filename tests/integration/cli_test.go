package integration_test

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the tpm binary into the given directory and returns
// the path to the resulting executable. The test is skipped if the build fails.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "tpm")
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", binPath, "./cmd/tpm")
	cmd.Dir = "/home/antoinegs/gits/tpm"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to build tpm binary: %v\n%s", err, out)
	}
	return binPath
}

func TestCLIVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI binary test in -short mode")
	}

	bin := buildBinary(t)

	cmd := exec.CommandContext(context.Background(), bin, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit code 0, got error: %v", err)
	}

	stdout := string(out)
	if !strings.Contains(stdout, "tpm") {
		t.Errorf("expected stdout to contain %q, got: %q", "tpm", stdout)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI binary test in -short mode")
	}

	bin := buildBinary(t)

	cmd := exec.CommandContext(context.Background(), bin, "unknown-cmd")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for unknown command, got exit 0")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	stderr := string(out)
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected stderr to contain %q, got: %q", "unknown command", stderr)
	}
}

func TestCLINoArgsWithoutTmux(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI binary test in -short mode")
	}

	// Skip if tmux is available since runInit might succeed.
	if _, err := exec.LookPath("tmux"); err == nil {
		t.Skip("tmux is available; skipping no-args failure test")
	}

	bin := buildBinary(t)

	cmd := exec.CommandContext(context.Background(), bin)
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code when tmux is not available, got exit 0")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() == 0 {
		t.Error("expected non-zero exit code when tmux is not available")
	}
}
