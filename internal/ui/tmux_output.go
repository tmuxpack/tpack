package ui

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tmux-plugins/tpm/internal/tmux"
)

// TmuxOutput displays messages via tmux run-shell echo.
type TmuxOutput struct {
	mu     sync.Mutex
	runner tmux.Runner
	failed atomic.Bool
}

// NewTmuxOutput returns a TmuxOutput using the given tmux runner.
func NewTmuxOutput(runner tmux.Runner) *TmuxOutput {
	return &TmuxOutput{runner: runner}
}

func (t *TmuxOutput) Ok(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_ = t.runner.RunShell("echo '" + shellEscapeSingleQuoted(msg) + "'")
}

func (t *TmuxOutput) Err(msg string) {
	t.failed.Store(true)
	t.mu.Lock()
	defer t.mu.Unlock()
	_ = t.runner.RunShell("echo '" + shellEscapeSingleQuoted(msg) + "'")
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

// shellEscapeSingleQuoted escapes s for safe use inside single quotes in a
// POSIX shell command. In single-quoted strings, only the single quote itself
// needs escaping (using the '\” break-and-rejoin technique).
// Null bytes are stripped as they can truncate shell arguments.
func shellEscapeSingleQuoted(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.ReplaceAll(s, "'", "'\\''")
}
