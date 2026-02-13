package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestApplyTheme_NilRunner(t *testing.T) {
	// Should not panic.
	ApplyTheme(nil)
}

func TestApplyTheme_FullTheme(t *testing.T) {
	// Save originals.
	origPrimary := primaryColor
	origSecondary := secondaryColor
	origAccent := accentColor
	origText := textColor
	t.Cleanup(func() {
		applyColors(origPrimary, origSecondary, origAccent, origText)
	})

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#aaaaaa,bg=#bbbbbb"
	m.Options["pane-active-border-style"] = "fg=#cccccc"
	m.Options["window-status-current-style"] = "bg=#dddddd"

	ApplyTheme(m)

	if primaryColor != lipgloss.Color("#bbbbbb") {
		t.Errorf("primaryColor = %q, want %q", primaryColor, "#bbbbbb")
	}
	if textColor != lipgloss.Color("#aaaaaa") {
		t.Errorf("textColor = %q, want %q", textColor, "#aaaaaa")
	}
	if secondaryColor != lipgloss.Color("#cccccc") {
		t.Errorf("secondaryColor = %q, want %q", secondaryColor, "#cccccc")
	}
	if accentColor != lipgloss.Color("#dddddd") {
		t.Errorf("accentColor = %q, want %q", accentColor, "#dddddd")
	}
}

func TestApplyTheme_PartialTheme(t *testing.T) {
	origPrimary := primaryColor
	origSecondary := secondaryColor
	origAccent := accentColor
	origText := textColor
	t.Cleanup(func() {
		applyColors(origPrimary, origSecondary, origAccent, origText)
	})

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#111111,bg=#222222"
	// pane-active-border-style and window-status-current-style not set.

	ApplyTheme(m)

	if primaryColor != lipgloss.Color("#222222") {
		t.Errorf("primaryColor = %q, want %q", primaryColor, "#222222")
	}
	if textColor != lipgloss.Color("#111111") {
		t.Errorf("textColor = %q, want %q", textColor, "#111111")
	}
	// secondaryColor and accentColor should keep defaults.
	if secondaryColor != origSecondary {
		t.Errorf("secondaryColor = %q, want original %q", secondaryColor, origSecondary)
	}
	if accentColor != origAccent {
		t.Errorf("accentColor = %q, want original %q", accentColor, origAccent)
	}
}

func TestApplyTheme_ErrorColor_Unchanged(t *testing.T) {
	origError := errorColor
	origMuted := mutedColor
	origPrimary := primaryColor
	origSecondary := secondaryColor
	origAccent := accentColor
	origText := textColor
	t.Cleanup(func() {
		applyColors(origPrimary, origSecondary, origAccent, origText)
	})

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#ffffff,bg=#000000"

	ApplyTheme(m)

	if errorColor != origError {
		t.Errorf("errorColor changed: %q, want %q", errorColor, origError)
	}
	if mutedColor != origMuted {
		t.Errorf("mutedColor changed: %q, want %q", mutedColor, origMuted)
	}
}

func TestApplyTheme_DefaultTmuxColors(t *testing.T) {
	origPrimary := primaryColor
	origSecondary := secondaryColor
	origAccent := accentColor
	origText := textColor
	t.Cleanup(func() {
		applyColors(origPrimary, origSecondary, origAccent, origText)
	})

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=default,bg=default"

	ApplyTheme(m)

	// "default" normalizes to "" → fallback to original.
	if primaryColor != origPrimary {
		t.Errorf("primaryColor = %q, want original %q", primaryColor, origPrimary)
	}
	if textColor != origText {
		t.Errorf("textColor = %q, want original %q", textColor, origText)
	}
}
