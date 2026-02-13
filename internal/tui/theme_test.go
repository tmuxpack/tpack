package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// saveAndRestore saves all 6 color variables and returns a cleanup function
// that restores them via applyColors.
func saveAndRestore(t *testing.T) {
	t.Helper()
	p, s, a, e, m, tx := primaryColor, secondaryColor, accentColor, errorColor, mutedColor, textColor
	t.Cleanup(func() {
		applyColors(p, s, a, e, m, tx)
	})
}

func TestApplyTheme_NilRunner(t *testing.T) {
	// Should not panic.
	ApplyTheme(nil)
}

func TestApplyTheme_FullTheme(t *testing.T) {
	saveAndRestore(t)

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
	saveAndRestore(t)
	origSecondary := secondaryColor
	origAccent := accentColor

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#111111,bg=#222222"

	ApplyTheme(m)

	if primaryColor != lipgloss.Color("#222222") {
		t.Errorf("primaryColor = %q, want %q", primaryColor, "#222222")
	}
	if textColor != lipgloss.Color("#111111") {
		t.Errorf("textColor = %q, want %q", textColor, "#111111")
	}
	if secondaryColor != origSecondary {
		t.Errorf("secondaryColor = %q, want original %q", secondaryColor, origSecondary)
	}
	if accentColor != origAccent {
		t.Errorf("accentColor = %q, want original %q", accentColor, origAccent)
	}
}

func TestApplyTheme_ErrorColor_Unchanged(t *testing.T) {
	saveAndRestore(t)
	origError := errorColor
	origMuted := mutedColor

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
	saveAndRestore(t)
	origPrimary := primaryColor
	origText := textColor

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=default,bg=default"

	ApplyTheme(m)

	if primaryColor != origPrimary {
		t.Errorf("primaryColor = %q, want original %q", primaryColor, origPrimary)
	}
	if textColor != origText {
		t.Errorf("textColor = %q, want original %q", textColor, origText)
	}
}

func TestApplyConfigColors_FullOverride(t *testing.T) {
	saveAndRestore(t)

	ApplyConfigColors(config.ColorConfig{
		Primary:   "#aa0000",
		Secondary: "#00bb00",
		Accent:    "#0000cc",
		Error:     "#dd0000",
		Muted:     "#555555",
		Text:      "#ffffff",
	})

	if primaryColor != lipgloss.Color("#aa0000") {
		t.Errorf("primaryColor = %q, want %q", primaryColor, "#aa0000")
	}
	if secondaryColor != lipgloss.Color("#00bb00") {
		t.Errorf("secondaryColor = %q, want %q", secondaryColor, "#00bb00")
	}
	if accentColor != lipgloss.Color("#0000cc") {
		t.Errorf("accentColor = %q, want %q", accentColor, "#0000cc")
	}
	if errorColor != lipgloss.Color("#dd0000") {
		t.Errorf("errorColor = %q, want %q", errorColor, "#dd0000")
	}
	if mutedColor != lipgloss.Color("#555555") {
		t.Errorf("mutedColor = %q, want %q", mutedColor, "#555555")
	}
	if textColor != lipgloss.Color("#ffffff") {
		t.Errorf("textColor = %q, want %q", textColor, "#ffffff")
	}
}

func TestApplyConfigColors_PartialOverride(t *testing.T) {
	saveAndRestore(t)
	origSecondary := secondaryColor
	origAccent := accentColor
	origError := errorColor
	origMuted := mutedColor
	origText := textColor

	ApplyConfigColors(config.ColorConfig{
		Primary: "#abcdef",
	})

	if primaryColor != lipgloss.Color("#abcdef") {
		t.Errorf("primaryColor = %q, want %q", primaryColor, "#abcdef")
	}
	if secondaryColor != origSecondary {
		t.Errorf("secondaryColor changed: %q, want %q", secondaryColor, origSecondary)
	}
	if accentColor != origAccent {
		t.Errorf("accentColor changed: %q, want %q", accentColor, origAccent)
	}
	if errorColor != origError {
		t.Errorf("errorColor changed: %q, want %q", errorColor, origError)
	}
	if mutedColor != origMuted {
		t.Errorf("mutedColor changed: %q, want %q", mutedColor, origMuted)
	}
	if textColor != origText {
		t.Errorf("textColor changed: %q, want %q", textColor, origText)
	}
}

func TestApplyConfigColors_EmptyNoOp(t *testing.T) {
	saveAndRestore(t)
	origPrimary := primaryColor
	origSecondary := secondaryColor
	origAccent := accentColor
	origError := errorColor
	origMuted := mutedColor
	origText := textColor

	ApplyConfigColors(config.ColorConfig{})

	if primaryColor != origPrimary {
		t.Errorf("primaryColor changed: %q, want %q", primaryColor, origPrimary)
	}
	if secondaryColor != origSecondary {
		t.Errorf("secondaryColor changed: %q, want %q", secondaryColor, origSecondary)
	}
	if accentColor != origAccent {
		t.Errorf("accentColor changed: %q, want %q", accentColor, origAccent)
	}
	if errorColor != origError {
		t.Errorf("errorColor changed: %q, want %q", errorColor, origError)
	}
	if mutedColor != origMuted {
		t.Errorf("mutedColor changed: %q, want %q", mutedColor, origMuted)
	}
	if textColor != origText {
		t.Errorf("textColor changed: %q, want %q", textColor, origText)
	}
}
