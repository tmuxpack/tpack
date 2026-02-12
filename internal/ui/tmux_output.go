package ui

import (
	"strings"
	"sync/atomic"

	"github.com/tmux-plugins/tpm/internal/tmux"
)

// TmuxOutput displays messages via tmux run-shell echo.
type TmuxOutput struct {
	runner tmux.Runner
	failed atomic.Bool
}

// NewTmuxOutput returns a TmuxOutput using the given tmux runner.
func NewTmuxOutput(runner tmux.Runner) *TmuxOutput {
	return &TmuxOutput{runner: runner}
}

func (t *TmuxOutput) Ok(msg string) {
	t.runner.RunShell("echo '" + escapeQuotes(msg) + "'")
}

func (t *TmuxOutput) Err(msg string) {
	t.failed.Store(true)
	t.runner.RunShell("echo '" + escapeQuotes(msg) + "'")
}

func (t *TmuxOutput) EndMessage() {
	continueKey := "ENTER"

	modeKeys, err := t.runner.ShowWindowOption("mode-keys")
	if err == nil && strings.Contains(modeKeys, "emacs") {
		continueKey = "ESCAPE"
	}

	t.Ok("")
	t.Ok("TMUX environment reloaded.")
	t.Ok("")
	t.Ok("Done, press " + continueKey + " to continue.")
}

func (t *TmuxOutput) HasFailed() bool {
	return t.failed.Load()
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}
