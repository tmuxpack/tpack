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
	t.Run("tmuxEcho false does not use RunShell", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(false, runner)
		out.Ok("probe")

		for _, call := range runner.Calls {
			if call.Method == "RunShell" {
				t.Errorf("newOutput(false, ...) made RunShell call: %+v", call)
			}
		}
	})

	t.Run("tmuxEcho true uses RunShell", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(true, runner)
		out.Ok("probe")

		if len(runner.Calls) != 1 || runner.Calls[0].Method != "RunShell" {
			t.Errorf("newOutput(true, ...) calls = %+v, want one RunShell", runner.Calls)
		}
	})
}
