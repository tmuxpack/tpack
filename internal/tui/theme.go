package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// BuildTheme queries the tmux server for the user's theme colors and
// returns a Theme. If runner is nil or a color cannot be detected,
// the hardcoded default is used.
func BuildTheme(runner tmux.Runner) Theme {
	base := DefaultTheme()
	if runner == nil {
		return base
	}

	tc := tmux.DetectTheme(runner)

	primary := base.PrimaryColor
	secondary := base.SecondaryColor
	accent := base.AccentColor
	text := base.TextColor

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

	return NewTheme(primary, secondary, accent, base.ErrorColor, base.MutedColor, text)
}

// OverlayConfigColors returns a new Theme with non-empty config color values
// applied on top of the given base theme.
func OverlayConfigColors(base Theme, cc config.ColorConfig) Theme {
	primary := base.PrimaryColor
	secondary := base.SecondaryColor
	accent := base.AccentColor
	errC := base.ErrorColor
	muted := base.MutedColor
	text := base.TextColor

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

	return NewTheme(primary, secondary, accent, errC, muted, text)
}
