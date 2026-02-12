package tmux_test

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/tmux"
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

	val, err := m.ShowOption("@foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
