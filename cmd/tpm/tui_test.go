package main

import "testing"

func TestPopupSupported(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"tmux 3.4 supports popup", "tmux 3.4", true},
		{"tmux 3.3a supports popup", "tmux 3.3a", true},
		{"tmux 3.2 supports popup", "tmux 3.2", true},
		{"tmux 3.2a supports popup", "tmux 3.2a", true},
		{"tmux 3.10 supports popup", "tmux 3.10", true},
		{"tmux 3.1 does not support popup", "tmux 3.1", false},
		{"tmux 3.0 does not support popup", "tmux 3.0", false},
		{"tmux 2.9 does not support popup", "tmux 2.9", false},
		{"tmux 1.9 does not support popup", "tmux 1.9", false},
		{"empty version string", "", false},
		{"unparseable version", "garbage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := popupSupported(tt.version)
			if got != tt.want {
				t.Errorf("popupSupported(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
