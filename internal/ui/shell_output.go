package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// ShellOutput writes messages to stdout/stderr for CLI usage.
type ShellOutput struct {
	mu     sync.Mutex
	stdout io.Writer
	stderr io.Writer
	failed atomic.Bool
}

// NewShellOutput returns a ShellOutput writing to os.Stdout/os.Stderr.
func NewShellOutput() *ShellOutput {
	return &ShellOutput{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// NewShellOutputWithWriters creates a ShellOutput with custom writers (for testing).
func NewShellOutputWithWriters(stdout, stderr io.Writer) *ShellOutput {
	return &ShellOutput{
		stdout: stdout,
		stderr: stderr,
	}
}

func (s *ShellOutput) Ok(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintln(s.stdout, msg)
}

func (s *ShellOutput) Err(msg string) {
	s.failed.Store(true)
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintln(s.stderr, msg)
}

func (s *ShellOutput) EndMessage() {
	// Shell output mode does not display an end message.
}

func (s *ShellOutput) HasFailed() bool {
	return s.failed.Load()
}
