package tmux_test

import (
	"errors"
	"testing"

	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestMockRunnerImplementsRunner(t *testing.T) {
	var _ tmux.Runner = (*tmux.MockRunner)(nil)
}

func TestRealRunnerImplementsRunner(t *testing.T) {
	var _ tmux.Runner = (*tmux.RealRunner)(nil)
}

func TestMockRunnerRecordsCalls(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["@foo"] = "bar"

	val, set, err := m.ShowOption("@foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !set {
		t.Fatal("option reported as unset")
	}
	if val != "bar" {
		t.Errorf("got %q, want %q", val, "bar")
	}
	if len(m.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(m.Calls))
	}
	if m.Calls[0].Method != "ShowOption" {
		t.Errorf("got method %q, want %q", m.Calls[0].Method, "ShowOption")
	}
}

func TestMockRunnerDistinguishesUnsetAndEmptyOptions(t *testing.T) {
	m := tmux.NewMockRunner()

	if value, set, err := m.ShowOption("@foo"); err != nil || set || value != "" {
		t.Fatalf("unset option = (%q, %v, %v), want empty, false, nil", value, set, err)
	}
	m.Options["@foo"] = ""
	if value, set, err := m.ShowOption("@foo"); err != nil || !set || value != "" {
		t.Fatalf("empty option = (%q, %v, %v), want empty, true, nil", value, set, err)
	}
}

func TestMockRunnerEnvironment(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Environment["FOO"] = "bar"

	val, err := m.ShowEnvironment("FOO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "bar" {
		t.Errorf("got %q, want %q", val, "bar")
	}
}

func TestMockRunnerSetEnvironment(t *testing.T) {
	m := tmux.NewMockRunner()

	if err := m.SetEnvironment("FOO", "baz"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Environment["FOO"] != "baz" {
		t.Errorf("expected environment FOO=baz, got %q", m.Environment["FOO"])
	}
}

func TestMockRunnerVersion(t *testing.T) {
	m := tmux.NewMockRunner()
	m.VersionStr = "tmux 3.4"

	v, err := m.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "tmux 3.4" {
		t.Errorf("got %q, want %q", v, "tmux 3.4")
	}
}

func TestMockRunnerExpandFormat(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Formats["#{config_files}"] = "/home/user/.tmux.conf,/custom/tmux.conf"

	got, err := m.ExpandFormat("#{config_files}")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/user/.tmux.conf,/custom/tmux.conf" {
		t.Fatalf("ExpandFormat() = %q", got)
	}
	if call := m.Calls[0]; call.Method != "ExpandFormat" || call.Args[0] != "#{config_files}" {
		t.Fatalf("call = %#v", call)
	}
}

func TestMockRunnerExpandFormatError(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Errors["ExpandFormat:#{host}"] = errors.New("no server")
	if _, err := m.ExpandFormat("#{host}"); err == nil {
		t.Fatal("expected format expansion error")
	}
}
