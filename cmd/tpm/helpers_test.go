package main

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{
			name: "flag present",
			args: []string{"--tmux-echo"},
			flag: "--tmux-echo",
			want: true,
		},
		{
			name: "flag absent",
			args: []string{"--shell-echo"},
			flag: "--tmux-echo",
			want: false,
		},
		{
			name: "empty args",
			args: []string{},
			flag: "--tmux-echo",
			want: false,
		},
		{
			name: "flag at end of multiple args",
			args: []string{"foo", "bar", "--tmux-echo"},
			flag: "--tmux-echo",
			want: true,
		},
		{
			name: "similar but different flag",
			args: []string{"--tmux-echo-extra"},
			flag: "--tmux-echo",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFlag(tt.args, tt.flag)
			if got != tt.want {
				t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	t.Run("no failure returns 0", func(t *testing.T) {
		output := ui.NewMockOutput()
		output.Ok("all good")

		if got := exitCode(output); got != 0 {
			t.Errorf("exitCode() = %d, want 0", got)
		}
	})

	t.Run("failure returns 1", func(t *testing.T) {
		output := ui.NewMockOutput()
		output.Err("something went wrong")

		if got := exitCode(output); got != 1 {
			t.Errorf("exitCode() = %d, want 1", got)
		}
	})
}

func TestNewOutput(t *testing.T) {
	t.Run("tmuxEcho false returns ShellOutput", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(false, runner)

		if _, ok := out.(*ui.ShellOutput); !ok {
			t.Errorf("newOutput(false, ...) returned %T, want *ui.ShellOutput", out)
		}
	})

	t.Run("tmuxEcho true returns TmuxOutput", func(t *testing.T) {
		runner := tmux.NewMockRunner()
		out := newOutput(true, runner)

		if _, ok := out.(*ui.TmuxOutput); !ok {
			t.Errorf("newOutput(true, ...) returned %T, want *ui.TmuxOutput", out)
		}
	})
}
