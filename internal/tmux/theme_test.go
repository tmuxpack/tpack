package tmux

import (
	"fmt"
	"testing"
)

func TestDetectTheme_FullTheme(t *testing.T) {
	m := NewMockRunner()
	m.Options["status-style"] = "fg=#ffffff,bg=#333333"
	m.Options["pane-active-border-style"] = "fg=#00ff00"
	m.Options["window-status-current-style"] = "fg=#ff0000,bg=#0000ff"

	tc := DetectTheme(m)

	if tc.Primary != "#333333" {
		t.Errorf("Primary = %q, want %q", tc.Primary, "#333333")
	}
	if tc.Text != "#ffffff" {
		t.Errorf("Text = %q, want %q", tc.Text, "#ffffff")
	}
	if tc.Secondary != "#00ff00" {
		t.Errorf("Secondary = %q, want %q", tc.Secondary, "#00ff00")
	}
	if tc.Accent != "#0000ff" {
		t.Errorf("Accent = %q, want %q (bg preferred over fg)", tc.Accent, "#0000ff")
	}
}

func TestDetectTheme_AccentFallsBackToFG(t *testing.T) {
	m := NewMockRunner()
	m.Options["window-status-current-style"] = "fg=#ff0000"

	tc := DetectTheme(m)

	if tc.Accent != "#ff0000" {
		t.Errorf("Accent = %q, want %q (should fall back to fg)", tc.Accent, "#ff0000")
	}
}

func TestDetectTheme_Colour256(t *testing.T) {
	m := NewMockRunner()
	m.Options["status-style"] = "fg=colour255,bg=colour234"
	m.Options["pane-active-border-style"] = "fg=colour82"
	m.Options["window-status-current-style"] = "bg=colour208"

	tc := DetectTheme(m)

	if tc.Primary != "234" {
		t.Errorf("Primary = %q, want %q", tc.Primary, "234")
	}
	if tc.Text != "255" {
		t.Errorf("Text = %q, want %q", tc.Text, "255")
	}
	if tc.Secondary != "82" {
		t.Errorf("Secondary = %q, want %q", tc.Secondary, "82")
	}
	if tc.Accent != "208" {
		t.Errorf("Accent = %q, want %q", tc.Accent, "208")
	}
}

func TestDetectTheme_DefaultColors(t *testing.T) {
	m := NewMockRunner()
	m.Options["status-style"] = "fg=default,bg=default"

	tc := DetectTheme(m)

	if tc.Primary != "" {
		t.Errorf("Primary = %q, want empty for default", tc.Primary)
	}
	if tc.Text != "" {
		t.Errorf("Text = %q, want empty for default", tc.Text)
	}
}

func TestDetectTheme_Errors(t *testing.T) {
	m := NewMockRunner()
	m.Errors["ShowOption:status-style"] = fmt.Errorf("not found")
	m.Errors["ShowOption:pane-active-border-style"] = fmt.Errorf("not found")
	m.Errors["ShowOption:window-status-current-style"] = fmt.Errorf("not found")

	tc := DetectTheme(m)

	if tc.Primary != "" || tc.Text != "" || tc.Secondary != "" || tc.Accent != "" {
		t.Errorf("expected all empty on errors, got %+v", tc)
	}
}

func TestDetectTheme_PartialTheme(t *testing.T) {
	m := NewMockRunner()
	m.Options["status-style"] = "fg=#aabbcc,bg=#ddeeff"
	// Other options not set — empty string from mock, no error.

	tc := DetectTheme(m)

	if tc.Primary != "#ddeeff" {
		t.Errorf("Primary = %q, want %q", tc.Primary, "#ddeeff")
	}
	if tc.Text != "#aabbcc" {
		t.Errorf("Text = %q, want %q", tc.Text, "#aabbcc")
	}
	if tc.Secondary != "" {
		t.Errorf("Secondary = %q, want empty", tc.Secondary)
	}
	if tc.Accent != "" {
		t.Errorf("Accent = %q, want empty", tc.Accent)
	}
}

func TestDetectTheme_EmptyOptions(t *testing.T) {
	m := NewMockRunner()
	// All options return empty string (default mock behavior).

	tc := DetectTheme(m)

	if tc.Primary != "" || tc.Text != "" || tc.Secondary != "" || tc.Accent != "" {
		t.Errorf("expected all empty for empty options, got %+v", tc)
	}
}
