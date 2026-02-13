package tmux

// ThemeColors holds the colors derived from the user's tmux theme.
// Empty fields indicate that no usable color was found and the caller
// should fall back to its own default.
type ThemeColors struct {
	Primary   string // from status-style bg
	Text      string // from status-style fg
	Secondary string // from pane-active-border-style fg
	Accent    string // from window-status-current-style bg (or fg)
}

// DetectTheme queries the tmux server for style options and returns
// normalized color values.  Errors from individual queries are silently
// ignored — the corresponding field is left empty so the caller can
// apply its fallback.
func DetectTheme(r Runner) ThemeColors {
	var tc ThemeColors

	if statusStyle, err := r.ShowOption("status-style"); err == nil {
		attrs := ParseStyle(statusStyle)
		tc.Primary = NormalizeColor(attrs.BG)
		tc.Text = NormalizeColor(attrs.FG)
	}

	if borderStyle, err := r.ShowOption("pane-active-border-style"); err == nil {
		attrs := ParseStyle(borderStyle)
		tc.Secondary = NormalizeColor(attrs.FG)
	}

	if windowStyle, err := r.ShowOption("window-status-current-style"); err == nil {
		attrs := ParseStyle(windowStyle)
		tc.Accent = NormalizeColor(attrs.BG)
		if tc.Accent == "" {
			tc.Accent = NormalizeColor(attrs.FG)
		}
	}

	return tc
}
