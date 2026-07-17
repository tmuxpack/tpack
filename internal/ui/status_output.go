package ui

import (
	"sync/atomic"

	"github.com/tmuxpack/tpack/internal/tmux"
)

// StatusOutput displays messages in the tmux status line via display-message.
// Every message is prefixed with "tpack:" (plus a level for Warn/Err) because
// the status line mixes output from many programs.
type StatusOutput struct {
	runner tmux.Runner
	failed atomic.Bool
}

// NewStatusOutput returns a StatusOutput using the given tmux runner.
func NewStatusOutput(runner tmux.Runner) *StatusOutput {
	return &StatusOutput{runner: runner}
}

func (s *StatusOutput) Ok(msg string) {
	_ = s.runner.DisplayMessage("tpack: " + msg)
}

func (s *StatusOutput) Warn(msg string) {
	_ = s.runner.DisplayMessage("tpack: warning: " + msg)
}

func (s *StatusOutput) Err(msg string) {
	s.failed.Store(true)
	_ = s.runner.DisplayMessage("tpack: error: " + msg)
}

// EndMessage is a no-op; the status line has no completion message.
func (s *StatusOutput) EndMessage() {}

func (s *StatusOutput) HasFailed() bool {
	return s.failed.Load()
}
