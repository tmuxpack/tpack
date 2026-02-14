package tmux_test

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestParseVersionDigits(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"tmux 1.9", 109},
		{"tmux 3.4", 304},
		{"1.9a", 109},
		{"tmux 3.3a", 303},
		{"tmux 2.1", 201},
		{"tmux 1.8", 108},
		{"3.0", 300},
		{"tmux next-3.4", 304},
		{"tmux 3.10", 310},
		{"tmux 10.0", 1000},
		{"", 0},
		{"no version here", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tmux.ParseVersionDigits(tt.input)
			if got != tt.want {
				t.Errorf("ParseVersionDigits(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVersionSupported(t *testing.T) {
	tests := []struct {
		name    string
		current int
		min     int
		want    bool
	}{
		{"exact match", 109, 109, true},
		{"newer", 304, 109, true},
		{"older", 108, 109, false},
		{"multi-digit minor newer", 310, 304, true},
		{"multi-digit minor older", 304, 310, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tmux.IsVersionSupported(tt.current, tt.min)
			if got != tt.want {
				t.Errorf("IsVersionSupported(%d, %d) = %v, want %v", tt.current, tt.min, got, tt.want)
			}
		})
	}
}
