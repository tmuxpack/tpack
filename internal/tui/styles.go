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
)

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
	contentWidth := m.width - 4 // BaseStyle horizontal padding (2 left + 2 right)
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
	contentWidth := m.width - 4
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

func renderScrollIndicators(start, end, total int) (string, string) {
	var top, bottom string
	if start > 0 {
		top = SubtitleStyle.Render("  ↑ more above") + "\n"
	}
	if end < total {
		bottom = SubtitleStyle.Render("  ↓ more below") + "\n"
	}
	return top, bottom
}
