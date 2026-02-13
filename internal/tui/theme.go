package tui

import (
	"github.com/charmbracelet/lipgloss"
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

	applyColors(primary, secondary, accent, text)
}
