package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// ApplyTheme queries the tmux server for the user's theme colors and
// rebuilds all TUI styles to match.  If runner is nil or a color cannot
// be detected, the hardcoded default is kept.
func ApplyTheme(runner tmux.Runner) {
	if runner == nil {
		return
	}

	tc := tmux.DetectTheme(runner)

	primary := primaryColor
	secondary := secondaryColor
	accent := accentColor
	text := textColor

	if tc.Primary != "" {
		primary = lipgloss.Color(tc.Primary)
	}
	if tc.Text != "" {
		text = lipgloss.Color(tc.Text)
	}
	if tc.Secondary != "" {
		secondary = lipgloss.Color(tc.Secondary)
	}
	if tc.Accent != "" {
		accent = lipgloss.Color(tc.Accent)
	}

	applyColors(primary, secondary, accent, errorColor, mutedColor, text)
}

// ApplyConfigColors overlays non-empty color values from the config file
// on top of the current colors. Must be called after ApplyTheme.
func ApplyConfigColors(cc config.ColorConfig) {
	primary := primaryColor
	secondary := secondaryColor
	accent := accentColor
	errC := errorColor
	muted := mutedColor
	text := textColor

	if cc.Primary != "" {
		primary = lipgloss.Color(cc.Primary)
	}
	if cc.Secondary != "" {
		secondary = lipgloss.Color(cc.Secondary)
	}
	if cc.Accent != "" {
		accent = lipgloss.Color(cc.Accent)
	}
	if cc.Error != "" {
		errC = lipgloss.Color(cc.Error)
	}
	if cc.Muted != "" {
		muted = lipgloss.Color(cc.Muted)
	}
	if cc.Text != "" {
		text = lipgloss.Color(cc.Text)
	}

	applyColors(primary, secondary, accent, errC, muted, text)
}
