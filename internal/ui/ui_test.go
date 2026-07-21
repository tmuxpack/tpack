package ui_test

import (
	"bytes"
	"errors"
	"reflect"
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
	if err := out.Result(); err != nil {
		t.Errorf("Result = %v, want nil", err)
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
	if !errors.Is(out.Result(), ui.ErrReported) {
		t.Errorf("Result = %v, want ErrReported", out.Result())
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
	if err := out.Result(); err != nil {
		t.Errorf("Result = %v, want nil", err)
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
	if err := out.Result(); err != nil {
		t.Errorf("Result = %v, want nil", err)
	}
}

func TestTmuxOutputErr(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Err("fail")

	if !errors.Is(out.Result(), ui.ErrReported) {
		t.Errorf("Result = %v, want ErrReported", out.Result())
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
	if want := "printf '%s\\n' 'warning: thing looks off'"; m.Calls[0].Args[0] != want {
		t.Errorf("arg = %q, want %q", m.Calls[0].Args[0], want)
	}
	if err := out.Result(); err != nil {
		t.Errorf("Result = %v, want nil", err)
	}
}

func TestTmuxOutputErrPrefix(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewTmuxOutput(m)

	out.Err("fail")

	if len(m.Calls) != 1 || m.Calls[0].Method != "RunShell" {
		t.Fatalf("expected 1 RunShell call, got %+v", m.Calls)
	}
	if want := "printf '%s\\n' 'error: fail'"; m.Calls[0].Args[0] != want {
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
	if !errors.Is(m.Result(), ui.ErrReported) {
		t.Errorf("Result = %v, want ErrReported", m.Result())
	}
}

func TestStatusOutputImplementsOutput(t *testing.T) {
	var _ ui.Output = (*ui.Reporter)(nil)
}

func TestStatusOutputLevels(t *testing.T) {
	tests := []struct {
		name string
		call func(o ui.Output)
		want string
	}{
		{"ok", func(o ui.Output) { o.Ok("3 updates available") }, "tpack: 3 updates available"},
		{"warn", func(o ui.Output) { o.Warn("repo sync failed") }, "tpack: warning: repo sync failed"},
		{"err", func(o ui.Output) { o.Err("self-update failed") }, "tpack: error: self-update failed"},
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

func TestStatusOutputResult(t *testing.T) {
	m := tmux.NewMockRunner()
	out := ui.NewStatusOutput(m)

	out.Ok("fine")
	out.Warn("meh")
	if err := out.Result(); err != nil {
		t.Errorf("Ok/Warn Result = %v, want nil", err)
	}
	out.Err("boom")
	if !errors.Is(out.Result(), ui.ErrReported) {
		t.Errorf("Err Result = %v, want ErrReported", out.Result())
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

func TestMultiOutputImplementsOutput(t *testing.T) {
	var _ ui.Output = (*ui.MultiOutput)(nil)
}

func TestMultiOutputFansOut(t *testing.T) {
	a, b := ui.NewMockOutput(), ui.NewMockOutput()
	out := ui.NewMultiOutput(a, b)

	out.Ok("o")
	out.Warn("w")
	out.Err("e")
	out.EndMessage()

	for i, m := range []*ui.MockOutput{a, b} {
		if len(m.OkMsgs) != 1 || m.OkMsgs[0] != "o" {
			t.Errorf("child %d OkMsgs = %v, want [o]", i, m.OkMsgs)
		}
		if len(m.WarnMsgs) != 1 || m.WarnMsgs[0] != "w" {
			t.Errorf("child %d WarnMsgs = %v, want [w]", i, m.WarnMsgs)
		}
		if len(m.ErrMsgs) != 1 || m.ErrMsgs[0] != "e" {
			t.Errorf("child %d ErrMsgs = %v, want [e]", i, m.ErrMsgs)
		}
		if m.EndCalls != 1 {
			t.Errorf("child %d EndCalls = %d, want 1", i, m.EndCalls)
		}
	}
}

func TestMultiOutputResult(t *testing.T) {
	healthy := ui.NewMockOutput()
	failed := ui.NewMockOutput()
	failed.Err("already broken")

	if err := ui.NewMultiOutput(healthy).Result(); err != nil {
		t.Errorf("healthy Result = %v, want nil", err)
	}
	if err := ui.NewMultiOutput(healthy, failed).Result(); !errors.Is(err, ui.ErrReported) {
		t.Errorf("failed Result = %v, want ErrReported", err)
	}
}

func TestReporterResult(t *testing.T) {
	sink := ui.NewMockSink()
	out := ui.NewReporter(sink)
	out.Warn("warning")
	if err := out.Result(); err != nil {
		t.Fatalf("warning result = %v", err)
	}
	out.Err("failure")
	if !errors.Is(out.Result(), ui.ErrReported) {
		t.Fatalf("result = %v, want ErrReported", out.Result())
	}
}

func TestReporterRecordsTransportFailure(t *testing.T) {
	sink := ui.NewMockSink()
	sink.Err = errors.New("tmux unavailable")
	out := ui.NewReporter(sink)
	out.Ok("hello")
	var transport *ui.TransportError
	if !errors.As(out.Result(), &transport) {
		t.Fatalf("result = %v, want TransportError", out.Result())
	}
}

func TestTmuxSinkEscapesFormats(t *testing.T) {
	runner := tmux.NewMockRunner()
	out := ui.NewReporter(ui.NewTmuxSink(runner))
	out.Ok("#(touch /tmp/pwned) #{pane_id} 'quoted'")
	want := `printf '%s\n' '##(touch /tmp/pwned) ##{pane_id} '\''quoted'\'''`
	if got := runner.Calls[0].Args[0]; got != want {
		t.Errorf("RunShell = %q, want %q", got, want)
	}
}

func TestStatusSinkEscapesFormats(t *testing.T) {
	runner := tmux.NewMockRunner()
	out := ui.NewReporter(ui.NewStatusSink(runner))
	out.Warn("#(command) #{pane_id}")
	if got, want := runner.Calls[0].Args[0], "tpack: warning: ##(command) ##{pane_id}"; got != want {
		t.Errorf("DisplayMessage = %q, want %q", got, want)
	}
}

func TestTmuxSinkLiteralCases(t *testing.T) {
	tests := []struct {
		name string
		call func(ui.Output)
		want string
	}{
		{name: "warning", call: func(o ui.Output) { o.Warn("-n") }, want: `printf '%s\n' 'warning: -n'`},
		{name: "error format", call: func(o ui.Output) { o.Err("#{pane_id}") }, want: `printf '%s\n' 'error: ##{pane_id}'`},
		{name: "newline and backslash", call: func(o ui.Output) { o.Ok("a\nb\\c") }, want: "printf '%s\\n' 'a\nb\\c'"},
		{name: "empty", call: func(o ui.Output) { o.Ok("") }, want: `printf '%s\n' ''`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tmux.NewMockRunner()
			out := ui.NewReporter(ui.NewTmuxSink(runner))
			tt.call(out)
			if got := runner.Calls[0].Args[0]; got != tt.want {
				t.Errorf("RunShell = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMultiSinkAttemptsAllChildren(t *testing.T) {
	first := ui.NewMockSink()
	first.Err = errors.New("first failed")
	second := ui.NewMockSink()
	out := ui.NewReporter(ui.NewMultiSink(first, second))
	out.Ok("message")
	if got := second.Texts(); !reflect.DeepEqual(got, []string{"message"}) {
		t.Errorf("second messages = %v", got)
	}
	var transport *ui.TransportError
	if !errors.As(out.Result(), &transport) {
		t.Fatalf("result = %v, want TransportError", out.Result())
	}
}

func TestSeveritySinkRoutesByLevel(t *testing.T) {
	info := ui.NewMockSink()
	warning := ui.NewMockSink()
	failure := ui.NewMockSink()
	out := ui.NewReporter(ui.NewSeveritySink(info, warning, failure))

	out.Ok("info")
	out.Warn("warning")
	out.Err("failure")
	out.EndMessage()

	if got := info.Texts(); !reflect.DeepEqual(got, []string{"info"}) {
		t.Errorf("info messages = %v, want [info]", got)
	}
	if got := warning.Texts(); !reflect.DeepEqual(got, []string{"warning"}) {
		t.Errorf("warning messages = %v, want [warning]", got)
	}
	if got := failure.Texts(); !reflect.DeepEqual(got, []string{"failure"}) {
		t.Errorf("failure messages = %v, want [failure]", got)
	}
	if info.EndCalls != 1 || warning.EndCalls != 0 || failure.EndCalls != 0 {
		t.Errorf("EndCalls = (%d, %d, %d), want (1, 0, 0)", info.EndCalls, warning.EndCalls, failure.EndCalls)
	}
}
