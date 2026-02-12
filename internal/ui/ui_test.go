package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestShellOutputOk(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := ui.NewShellOutputWithWriters(&stdout, &stderr)

	out.Ok("hello")

	if got := stdout.String(); got != "hello\n" {
		t.Errorf("stdout = %q, want %q", got, "hello\n")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty, got %q", stderr.String())
	}
	if out.HasFailed() {
		t.Error("HasFailed should be false")
	}
}

func TestShellOutputErr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := ui.NewShellOutputWithWriters(&stdout, &stderr)

	out.Err("fail msg")

	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty, got %q", stdout.String())
	}
	if got := stderr.String(); got != "fail msg\n" {
		t.Errorf("stderr = %q, want %q", got, "fail msg\n")
	}
	if !out.HasFailed() {
		t.Error("HasFailed should be true")
	}
}

func TestShellOutputEndMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := ui.NewShellOutputWithWriters(&stdout, &stderr)

	out.EndMessage()

	if stdout.Len() != 0 {
		t.Error("shell EndMessage should not produce output")
	}
}

func TestTmuxOutputOk(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Ok("hello")

	if len(m.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(m.Calls))
	}
	if m.Calls[0].Method != "RunShell" {
		t.Errorf("expected RunShell call, got %s", m.Calls[0].Method)
	}
	if out.HasFailed() {
		t.Error("HasFailed should be false after Ok")
	}
}

func TestTmuxOutputErr(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Err("fail")

	if !out.HasFailed() {
		t.Error("HasFailed should be true")
	}
}

func TestTmuxOutputEndMessageVI(t *testing.T) {
	m := tmux.NewMockRunner()
	m.WindowOpts["mode-keys"] = "mode-keys vi"
	out := ui.NewTmuxOutput(m)

	out.EndMessage()

	foundEnter := false
	for _, c := range m.Calls {
		if c.Method == "RunShell" {
			for _, arg := range c.Args {
				if strings.Contains(arg, "ENTER") {
					foundEnter = true
				}
			}
		}
	}
	if !foundEnter {
		t.Error("vi mode should show ENTER")
	}
}

func TestTmuxOutputEndMessageEmacs(t *testing.T) {
	m := tmux.NewMockRunner()
	m.WindowOpts["mode-keys"] = "mode-keys emacs"
	out := ui.NewTmuxOutput(m)

	out.EndMessage()

	foundEscape := false
	for _, c := range m.Calls {
		if c.Method == "RunShell" {
			for _, arg := range c.Args {
				if strings.Contains(arg, "ESCAPE") {
					foundEscape = true
				}
			}
		}
	}
	if !foundEscape {
		t.Error("emacs mode should show ESCAPE")
	}
}

func TestMockOutputImplementsOutput(t *testing.T) {
	var _ ui.Output = (*ui.MockOutput)(nil)
}

func TestMockOutput(t *testing.T) {
	m := ui.NewMockOutput()
	m.Ok("a")
	m.Ok("b")
	m.Err("c")
	m.EndMessage()

	if len(m.OkMsgs) != 2 {
		t.Errorf("expected 2 Ok msgs, got %d", len(m.OkMsgs))
	}
	if len(m.ErrMsgs) != 1 {
		t.Errorf("expected 1 Err msg, got %d", len(m.ErrMsgs))
	}
	if m.EndCalls != 1 {
		t.Errorf("expected 1 EndMessage call, got %d", m.EndCalls)
	}
	if !m.HasFailed() {
		t.Error("HasFailed should be true")
	}
}
