package main

import (
	"errors"
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

func TestNewCommandOutput(t *testing.T) {
	t.Run("tmuxEcho false does not use RunShell", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newCommandOutput(false, runner)
		out.Ok("probe")

		for _, call := range runner.Calls {
			if call.Method == "RunShell" {
				t.Errorf("newCommandOutput(false, ...) made RunShell call: %+v", call)
			}
		}
	})

	t.Run("tmuxEcho true uses RunShell", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newCommandOutput(true, runner)
		out.Ok("probe")

		if len(runner.Calls) != 1 || runner.Calls[0].Method != "RunShell" {
			t.Errorf("newCommandOutput(true, ...) calls = %+v, want one RunShell", runner.Calls)
		}
	})
}

func TestOutputResultReturnsTransportFailure(t *testing.T) {
	sink := ui.NewMockSink()
	sink.Err = errors.New("tmux unavailable")
	out := ui.NewReporter(sink)
	out.Err("already reported")
	err := outputResult(out)
	var transport *ui.TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("outputResult = %v, want transport error", err)
	}
}

func TestOutputResultUsesErrSilentWhenDelivered(t *testing.T) {
	out := ui.NewReporter(ui.NewMockSink())
	out.Err("already reported")
	if err := outputResult(out); !errors.Is(err, errSilent) {
		t.Fatalf("outputResult = %v, want errSilent", err)
	}
}
