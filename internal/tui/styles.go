package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds all TUI styles derived from a color palette.
type Theme struct {
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	AccentColor    lipgloss.Color
	ErrorColor     lipgloss.Color
	MutedColor     lipgloss.Color
	TextColor      lipgloss.Color

	BaseStyle               lipgloss.Style
	TitleStyle              lipgloss.Style
	SubtitleStyle           lipgloss.Style
	MutedTextStyle          lipgloss.Style
	SelectedRowStyle        lipgloss.Style
	CheckedStyle            lipgloss.Style
	UncheckedStyle          lipgloss.Style
	StatusInstalledStyle    lipgloss.Style
	StatusNotInstalledStyle lipgloss.Style
	StatusOutdatedStyle     lipgloss.Style
	StatusCheckFailedStyle  lipgloss.Style
	SuccessStyle            lipgloss.Style
	ErrorStyle              lipgloss.Style
	HelpStyle               lipgloss.Style
	HelpKeyStyle            lipgloss.Style
	ProgressStyle           lipgloss.Style
	OrphanStyle             lipgloss.Style
}

// NewTheme constructs a Theme from the given color palette.
func NewTheme(primary, secondary, accent, errC, muted, text lipgloss.Color) Theme {
	return Theme{
		PrimaryColor:   primary,
		SecondaryColor: secondary,
		AccentColor:    accent,
		ErrorColor:     errC,
		MutedColor:     muted,
		TextColor:      text,

		BaseStyle: lipgloss.NewStyle().
			Padding(1, 2),

		TitleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			MarginBottom(1).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary),

		SubtitleStyle: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true).
			MarginBottom(1),

		MutedTextStyle: lipgloss.NewStyle().
			Foreground(muted),

		SelectedRowStyle: lipgloss.NewStyle().
			Foreground(text).
			Background(primary).
			Bold(true),

		CheckedStyle: lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true),

		UncheckedStyle: lipgloss.NewStyle().
			Foreground(muted),

		StatusInstalledStyle: lipgloss.NewStyle().
			Foreground(secondary),

		StatusNotInstalledStyle: lipgloss.NewStyle().
			Foreground(errC),

		StatusOutdatedStyle: lipgloss.NewStyle().
			Foreground(accent),

		StatusCheckFailedStyle: lipgloss.NewStyle().
			Foreground(accent),

		SuccessStyle: lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true),

		ErrorStyle: lipgloss.NewStyle().
			Foreground(errC).
			Bold(true),

		HelpStyle: lipgloss.NewStyle().
			Foreground(muted).
			MarginTop(1),

		HelpKeyStyle: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		ProgressStyle: lipgloss.NewStyle().
			Foreground(secondary),

		OrphanStyle: lipgloss.NewStyle().
			Foreground(accent).
			Italic(true),
	}
}

// DefaultTheme returns a Theme built from the hardcoded default colors.
func DefaultTheme() Theme {
	return NewTheme(
		lipgloss.Color("#7C3AED"),
		lipgloss.Color("#10B981"),
		lipgloss.Color("#F59E0B"),
		lipgloss.Color("#EF4444"),
		lipgloss.Color("#6B7280"),
		lipgloss.Color("#F3F4F6"),
	)
}

func (th *Theme) renderHelp(width int, keys ...string) string {
	if width < 20 {
		width = 20
	}

	var lines []string
	var lineTexts []string
	var currentVisibleLen int
	separator := "  "
	separatorLen := len(separator)

	for i := 0; i < len(keys); i += 2 {
		key := keys[i]
		desc := ""
		if i+1 < len(keys) {
			desc = keys[i+1]
		}

		itemText := th.HelpKeyStyle.Render(key) + " " + desc
		itemLen := len(key) + 1 + len(desc)

		willExceed := len(lineTexts) > 0 && currentVisibleLen+separatorLen+itemLen > width-4

		if willExceed {
			lines = append(lines, strings.Join(lineTexts, separator))
			lineTexts = []string{itemText}
			currentVisibleLen = itemLen
		} else {
			lineTexts = append(lineTexts, itemText)
			if len(lineTexts) > 1 {
				currentVisibleLen += separatorLen
			}
			currentVisibleLen += itemLen
		}
	}

	if len(lineTexts) > 0 {
		lines = append(lines, strings.Join(lineTexts, separator))
	}

	return th.HelpStyle.Render(strings.Join(lines, "\n"))
}

// centerText centers a single-line element within the available content width.
// Returns text unchanged when the real window size is not yet known (e.g. during IdealSize).
func (m *Model) centerText(text string) string {
	if !m.sizeKnown || m.width <= 0 {
		return text
	}
	contentWidth := m.width - BaseStylePadding
	if lipgloss.Width(text) >= contentWidth {
		return text
	}
	return lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, text)
}

// centerBlock centers a multi-line block while preserving internal alignment.
// All lines are first padded to the widest line's width so they share the same left edge,
// then the uniform block is centered within the available content width.
func (m *Model) centerBlock(block string) string {
	if !m.sizeKnown || m.width <= 0 {
		return block
	}
	contentWidth := m.width - BaseStylePadding
	blockWidth := lipgloss.Width(block)
	if blockWidth >= contentWidth {
		return block
	}
	// Pad all lines to the same width to preserve internal alignment.
	aligned := lipgloss.NewStyle().Width(blockWidth).Render(block)
	return lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, aligned)
}

func renderCursor(active bool) string {
	if active {
		return "> "
	}
	return "  "
}

func (th *Theme) renderCheckbox(checked bool) string {
	if checked {
		return th.CheckedStyle.Render("[✓]")
	}
	return th.UncheckedStyle.Render("[ ]")
}

// padToBottom inserts vertical padding between body and footer so the footer
// is pinned to the bottom of the available content area.
func padToBottom(body, footer string, height int) string {
	contentHeight := height - BaseStyleVerticalPadding
	bodyHeight := lipgloss.Height(body)
	footerHeight := lipgloss.Height(footer)
	padding := contentHeight - bodyHeight - footerHeight
	if padding < 1 {
		padding = 1
	}
	return body + strings.Repeat("\n", padding) + footer
}

func (th *Theme) renderScrollIndicators(start, end, total int) (top, bottom string, dataStart, dataEnd int) {
	dataStart = start
	dataEnd = end
	if start > 0 {
		top = th.MutedTextStyle.Render("  ↑ more above") + "\n"
		dataStart++
	}
	if end < total {
		bottom = th.MutedTextStyle.Render("  ↓ more below") + "\n"
		dataEnd--
	}
	return top, bottom, dataStart, dataEnd
}
