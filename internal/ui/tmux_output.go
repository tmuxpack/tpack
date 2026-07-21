package ui

import (
	"errors"
	"strings"
	"sync"

	"github.com/tmuxpack/tpack/internal/shell"
)

func tmuxLiteral(text string) string { return strings.ReplaceAll(text, "#", "##") }

type tmuxPaneRunner interface {
	RunShell(string) error
	ShowWindowOption(string) (string, error)
}

type tmuxSink struct {
	mu     sync.Mutex
	runner tmuxPaneRunner
}

// NewTmuxSink returns a sink that writes literal messages to a tmux pane.
func NewTmuxSink(runner tmuxPaneRunner) Sink {
	return &tmuxSink{runner: runner}
}

func (s *tmuxSink) Write(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := ""
	switch message.Level {
	case LevelWarning:
		prefix = "warning: "
	case LevelError:
		prefix = "error: "
	case LevelInfo:
	default:
	}
	text := tmuxLiteral(prefix + message.Text)
	return s.runner.RunShell("printf '%s\\n' " + shell.Quote(text))
}

func (s *tmuxSink) EndMessage() error {
	continueKey := "ENTER"
	if modeKeys, err := s.runner.ShowWindowOption("mode-keys"); err == nil && strings.Contains(modeKeys, "emacs") {
		continueKey = "ESCAPE"
	}
	return errors.Join(
		s.Write(Message{Level: LevelInfo, Text: ""}),
		s.Write(Message{Level: LevelInfo, Text: "TMUX environment reloaded."}),
		s.Write(Message{Level: LevelInfo, Text: ""}),
		s.Write(Message{Level: LevelInfo, Text: "Done, press " + continueKey + " to continue."}),
	)
}

// NewTmuxOutput returns a Reporter using the given tmux pane runner.
func NewTmuxOutput(runner tmuxPaneRunner) *Reporter {
	return NewReporter(NewTmuxSink(runner))
}
