package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")
	textColor      = lipgloss.Color("#F3F4F6")

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
)

func init() {
	applyColors(primaryColor, secondaryColor, accentColor, errorColor, mutedColor, textColor)
}

// applyColors rebuilds every style variable from the given colors.
// It is called once at init time with defaults and may be called again
// after theme detection to apply tmux-derived colors.
// Must be called before the TUI event loop starts; not safe for concurrent use.
func applyColors(primary, secondary, accent, errC, muted, text lipgloss.Color) {
	primaryColor = primary
	secondaryColor = secondary
	accentColor = accent
	errorColor = errC
	mutedColor = muted
	textColor = text

	BaseStyle = lipgloss.NewStyle().
		Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		MarginBottom(1).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor)

	SubtitleStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		MarginBottom(1)

	MutedTextStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	SelectedRowStyle = lipgloss.NewStyle().
		Foreground(textColor).
		Background(primaryColor).
		Bold(true)

	CheckedStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)

	UncheckedStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	StatusInstalledStyle = lipgloss.NewStyle().
		Foreground(secondaryColor)

	StatusNotInstalledStyle = lipgloss.NewStyle().
		Foreground(errorColor)

	StatusOutdatedStyle = lipgloss.NewStyle().
		Foreground(accentColor)

	StatusCheckFailedStyle = lipgloss.NewStyle().
		Foreground(accentColor)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	HelpStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	ProgressStyle = lipgloss.NewStyle().
		Foreground(secondaryColor)

	OrphanStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Italic(true)
}

func renderHelp(width int, keys ...string) string {
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

		itemText := HelpKeyStyle.Render(key) + " " + desc
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

	return HelpStyle.Render(strings.Join(lines, "\n"))
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

func renderCheckbox(checked bool) string {
	if checked {
		return CheckedStyle.Render("[✓]")
	}
	return UncheckedStyle.Render("[ ]")
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

func renderScrollIndicators(start, end, total int) (top, bottom string, dataStart, dataEnd int) {
	dataStart = start
	dataEnd = end
	if start > 0 {
		top = MutedTextStyle.Render("  ↑ more above") + "\n"
		dataStart++
	}
	if end < total {
		bottom = MutedTextStyle.Render("  ↓ more below") + "\n"
		dataEnd--
	}
	return top, bottom, dataStart, dataEnd
}
