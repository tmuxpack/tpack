package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
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
	if got := stderr.String(); got != "tpack: error: fail msg\n" {
		t.Errorf("stderr = %q, want %q", got, "tpack: error: fail msg\n")
	}
	if !out.HasFailed() {
		t.Error("HasFailed should be true")
	}
}

func TestShellOutputWarn(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := ui.NewShellOutputWithWriters(&stdout, &stderr)

	out.Warn("disk almost full")

	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty, got %q", stdout.String())
	}
	if got := stderr.String(); got != "tpack: warning: disk almost full\n" {
		t.Errorf("stderr = %q, want %q", got, "tpack: warning: disk almost full\n")
	}
	if out.HasFailed() {
		t.Error("Warn must not mark output as failed")
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

func TestTmuxOutputWarnPrefix(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Warn("thing looks off")

	if len(m.Calls) != 1 || m.Calls[0].Method != "RunShell" {
		t.Fatalf("expected 1 RunShell call, got %+v", m.Calls)
	}
	if want := "echo 'warning: thing looks off'"; m.Calls[0].Args[0] != want {
		t.Errorf("arg = %q, want %q", m.Calls[0].Args[0], want)
	}
	if out.HasFailed() {
		t.Error("Warn must not mark output as failed")
	}
}

func TestTmuxOutputErrPrefix(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Err("fail")

	if len(m.Calls) != 1 || m.Calls[0].Method != "RunShell" {
		t.Fatalf("expected 1 RunShell call, got %+v", m.Calls)
	}
	if want := "echo 'error: fail'"; m.Calls[0].Args[0] != want {
		t.Errorf("arg = %q, want %q", m.Calls[0].Args[0], want)
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

func TestStatusOutputImplementsOutput(t *testing.T) {
	var _ ui.Output = (*ui.StatusOutput)(nil)
}

func TestStatusOutputLevels(t *testing.T) {
	tests := []struct {
		name string
		call func(o *ui.StatusOutput)
		want string
	}{
		{"ok", func(o *ui.StatusOutput) { o.Ok("3 updates available") }, "tpack: 3 updates available"},
		{"warn", func(o *ui.StatusOutput) { o.Warn("repo sync failed") }, "tpack: warning: repo sync failed"},
		{"err", func(o *ui.StatusOutput) { o.Err("self-update failed") }, "tpack: error: self-update failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			out := ui.NewStatusOutput(m)
			tt.call(out)
			if len(m.Calls) != 1 || m.Calls[0].Method != "DisplayMessage" {
				t.Fatalf("expected 1 DisplayMessage call, got %+v", m.Calls)
			}
			if m.Calls[0].Args[0] != tt.want {
				t.Errorf("msg = %q, want %q", m.Calls[0].Args[0], tt.want)
			}
		})
	}
}

func TestStatusOutputHasFailed(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewStatusOutput(m)

	out.Ok("fine")
	out.Warn("meh")
	if out.HasFailed() {
		t.Error("Ok/Warn must not mark output as failed")
	}
	out.Err("boom")
	if !out.HasFailed() {
		t.Error("Err must mark output as failed")
	}
}

func TestStatusOutputEndMessageNoop(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewStatusOutput(m)

	out.EndMessage()

	if len(m.Calls) != 0 {
		t.Errorf("EndMessage must not call tmux, got %+v", m.Calls)
	}
}
