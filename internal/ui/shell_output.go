package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type shellSink struct {
	mu     sync.Mutex
	stdout io.Writer
	stderr io.Writer
}

// NewShellSink returns a sink that writes informational messages to stdout and
// warnings and errors to stderr.
func NewShellSink(stdout, stderr io.Writer) Sink {
	return &shellSink{stdout: stdout, stderr: stderr}
}

func (s *shellSink) Write(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch message.Level {
	case LevelWarning:
		_, err := fmt.Fprintln(s.stderr, "tpack: warning: "+message.Text)
		return err
	case LevelError:
		_, err := fmt.Fprintln(s.stderr, "tpack: error: "+message.Text)
		return err
	case LevelInfo:
		_, err := fmt.Fprintln(s.stdout, message.Text)
		return err
	default:
		_, err := fmt.Fprintln(s.stdout, message.Text)
		return err
	}
}

func (s *shellSink) EndMessage() error { return nil }

// NewShellOutput returns a Reporter writing to os.Stdout and os.Stderr.
func NewShellOutput() *Reporter {
	return NewReporter(NewShellSink(os.Stdout, os.Stderr))
}

// NewShellOutputWithWriters returns a Reporter with custom writers.
func NewShellOutputWithWriters(stdout, stderr io.Writer) *Reporter {
	return NewReporter(NewShellSink(stdout, stderr))
}
