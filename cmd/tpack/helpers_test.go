package main

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

func TestExitCode(t *testing.T) {
	t.Run("no failure returns 0", func(t *testing.T) {
		output := ui.NewMockOutput()
		output.Ok("all good")

		if got := exitCode(output); got != 0 {
			t.Errorf("exitCode() = %d, want 0", got)
		}
	})

	t.Run("failure returns 1", func(t *testing.T) {
		output := ui.NewMockOutput()
		output.Err("something went wrong")

		if got := exitCode(output); got != 1 {
			t.Errorf("exitCode() = %d, want 1", got)
		}
	})
}

func TestNewOutput(t *testing.T) {
	t.Run("tmuxEcho false returns ShellOutput", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(false, runner)

		if _, ok := out.(*ui.ShellOutput); !ok {
			t.Errorf("newOutput(false, ...) returned %T, want *ui.ShellOutput", out)
		}
	})

	t.Run("tmuxEcho true returns TmuxOutput", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(true, runner)

		if _, ok := out.(*ui.TmuxOutput); !ok {
			t.Errorf("newOutput(true, ...) returned %T, want *ui.TmuxOutput", out)
		}
	})
}
