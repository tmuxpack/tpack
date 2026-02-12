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
		{"tmux 1.9", 19},
		{"tmux 3.4", 34},
		{"1.9a", 19},
		{"tmux 3.3a", 33},
		{"tmux 2.1", 21},
		{"tmux 1.8", 18},
		{"3.0", 30},
		{"tmux next-3.4", 34},
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
		{"exact match", 19, 19, true},
		{"newer", 34, 19, true},
		{"older", 18, 19, false},
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
