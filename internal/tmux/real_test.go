package tmux_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestRealRunnerCommandError(t *testing.T) {
	for _, tt := range []struct {
		name   string
		script string
		detail string
	}{
		{
			name:   "stderr",
			script: "printf '  error connecting to socket (No such file or directory)\\n' >&2\nexit 1\n",
			detail: "error connecting to socket (No such file or directory)",
		},
		{
			name:   "stdout",
			script: "printf 'command failed\\n'\nexit 1\n",
			detail: "command failed",
		},
		{
			name:   "no output",
			script: "exit 1\n",
		},
		{
			name:   "whitespace output",
			script: "printf ' \\n\\t' >&2\nexit 1\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			installTmuxScript(t, tt.script)
			runner := &tmux.RealRunner{}
			err := runner.SetEnvironment("TPACK_PLUGIN_PATH", "/plugins/")
			if err == nil {
				t.Fatal("expected command failure")
			}
			for _, want := range []string{"tmux set-environment", "exit status 1", tt.detail} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if strings.TrimSpace(err.Error()) != err.Error() || strings.HasSuffix(err.Error(), ":") {
				t.Errorf("error has trailing whitespace or empty detail: %q", err)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected wrapped *exec.ExitError, got %T: %v", err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
			}
		})
	}
}

func TestRealRunnerMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	runner := &tmux.RealRunner{}
	err := runner.SetEnvironment("TPACK_PLUGIN_PATH", "/plugins/")
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("expected wrapped exec.ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "tmux set-environment") {
		t.Errorf("error lacks command context: %v", err)
	}
}

func TestRealRunnerSuccessfulOutput(t *testing.T) {
	installTmuxScript(t, "printf '  tmux 3.8\\n'\n")
	runner := &tmux.RealRunner{}
	got, err := runner.Version()
	if err != nil {
		t.Fatalf("Version() failed: %v", err)
	}
	if got != "tmux 3.8" {
		t.Errorf("Version() = %q, want %q", got, "tmux 3.8")
	}
}

func installTmuxScript(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}
