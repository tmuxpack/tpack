package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/shell"
	"github.com/tmuxpack/tpack/internal/ui"
)

type paneRunner struct{ socket string }

func (r paneRunner) RunShell(command string) error {
	return exec.CommandContext(context.Background(), "tmux", "-S", r.socket, "run-shell", command).Run()
}

func (r paneRunner) ShowWindowOption(string) (string, error) { return "vi", nil }

func TestTmuxOutputTreatsFormatCommandAsLiteral(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found")
	}
	socket := filepath.Join(t.TempDir(), "tmux.sock")
	start := exec.CommandContext(context.Background(), "tmux", "-S", socket, "-f", "/dev/null", "new-session", "-d")
	if out, err := start.CombinedOutput(); err != nil {
		t.Fatalf("start tmux: %v: %s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.CommandContext(context.Background(), "tmux", "-S", socket, "kill-server").Run()
	})

	marker := filepath.Join(t.TempDir(), "format-command-ran")
	payload := "#(touch " + shell.Quote(marker) + ")"
	output := ui.NewTmuxOutput(paneRunner{socket: socket})
	output.Ok(payload)
	if err := output.Result(); err != nil {
		t.Fatalf("output result: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("tmux executed format command; marker stat error = %v", err)
	}
}
