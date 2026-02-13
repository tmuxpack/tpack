package tmux

import (
	"strings"
)

// StyleAttrs holds the foreground and background colors parsed from a tmux
// style string (e.g. "fg=#aabbcc,bg=colour123,bold").
type StyleAttrs struct {
	FG string
	BG string
}

// ParseStyle extracts foreground and background color values from a tmux
// style string.  Attributes like "bold" and "italics" are ignored.
// Examples:
//
//	"fg=#aabbcc,bg=colour123,bold" → StyleAttrs{FG: "#aabbcc", BG: "colour123"}
//	"bg=red"                       → StyleAttrs{BG: "red"}
func ParseStyle(style string) StyleAttrs {
	var attrs StyleAttrs
	for _, part := range strings.Split(style, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "fg="):
			attrs.FG = strings.TrimPrefix(part, "fg=")
		case strings.HasPrefix(part, "bg="):
			attrs.BG = strings.TrimPrefix(part, "bg=")
		}
	}
	return attrs
}

// NormalizeColor converts a tmux color value into a form suitable for
// lipgloss.Color().
//
//   - "#aabbcc"      → "#aabbcc"  (hex passthrough)
//   - "colour123"    → "123"      (256-color index)
//   - "color123"     → "123"      (alternate spelling)
//   - "red", "blue"  → "red", "blue" (named colors passthrough)
//   - "default", ""  → ""         (no color)
func NormalizeColor(tmuxColor string) string {
	tmuxColor = strings.TrimSpace(tmuxColor)
	if tmuxColor == "" || tmuxColor == "default" {
		return ""
	}
	if strings.HasPrefix(tmuxColor, "colour") { //nolint:misspell // tmux uses British spelling
		return strings.TrimPrefix(tmuxColor, "colour") //nolint:misspell // tmux uses British spelling
	}
	if strings.HasPrefix(tmuxColor, "color") {
		return strings.TrimPrefix(tmuxColor, "color")
	}
	return tmuxColor
}
