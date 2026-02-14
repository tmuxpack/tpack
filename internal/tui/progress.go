package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
)

// newProgress creates a configured progress bar.
func newProgress() progress.Model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)
	return p
}

// viewProgress renders the progress screen.
func (m *Model) viewProgress() string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf("  %s in progress...  ", m.operation)
	b.WriteString(m.centerText(m.theme.TitleStyle.Render(title)))
	b.WriteString("\n")

	// Counter
	counter := fmt.Sprintf("Processing %d of %d plugins", m.completedItems, m.totalItems)
	b.WriteString(m.centerText(m.theme.SubtitleStyle.Render(counter)))
	b.WriteString("\n\n")

	// Current item
	if len(m.inFlightNames) > 0 {
		current := "Current: " + strings.Join(m.inFlightNames, ", ")
		b.WriteString(m.centerText(current))
		b.WriteString("\n\n")
	}

	// Progress bar
	percent := float64(0)
	if m.totalItems > 0 {
		percent = float64(m.completedItems) / float64(m.totalItems)
	}
	b.WriteString(m.centerText(m.progressBar.ViewAs(percent)))
	b.WriteString("\n\n")

	// Stats
	successCount := 0
	failCount := 0
	for _, r := range m.results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	stats := fmt.Sprintf("%s %d successful  %s %d failed",
		m.theme.SuccessStyle.Render("✓"), successCount,
		m.theme.ErrorStyle.Render("✗"), failCount,
	)
	b.WriteString(m.centerText(m.theme.MutedTextStyle.Render(stats)))

	// Show results detail if complete
	if !m.processing && len(m.results) > 0 {
		b.WriteString("\n\n")
		b.WriteString(m.centerBlock(m.renderResults()))

		helpKeys := []string{"enter", "view commits"}
		if m.autoOp != OpNone {
			helpKeys = append(helpKeys, "q", "quit")
		} else {
			helpKeys = append(helpKeys, "q", "quit", "esc", "back to list")
		}
		help := m.centerText(m.theme.renderHelp(m.width, helpKeys...))
		return padToBottom(b.String(), help, m.height)
	}

	return b.String()
}

// renderResults renders the completed results list with cursor, commit counts, and scrolling.
func (m *Model) renderResults() string {
	viewHeight := m.resultMaxVisible()
	if viewHeight <= 0 {
		viewHeight = len(m.results)
	}
	start, end := calculateVisibleRange(m.resultScroll.scrollOffset, viewHeight, len(m.results))
	topIndicator, bottomIndicator, dataStart, dataEnd := m.theme.renderScrollIndicators(start, end, len(m.results))

	var rb strings.Builder
	rb.WriteString(topIndicator)
	for i := dataStart; i < dataEnd; i++ {
		r := m.results[i]
		cursor := "  "
		if i == m.resultScroll.cursor {
			cursor = "> "
		}
		if r.Success {
			rb.WriteString(renderSuccessResult(&m.theme, cursor, r))
		} else {
			rb.WriteString(cursor + "  " + m.theme.ErrorStyle.Render("✗ "+r.Name+": "+r.Message))
		}
		rb.WriteString("\n")
	}
	rb.WriteString(bottomIndicator)
	return strings.TrimRight(rb.String(), "\n")
}

// renderSuccessResult renders a single successful result line with commit count and indicator.
func renderSuccessResult(th *Theme, cursor string, r ResultItem) string {
	commitInfo := ""
	if n := len(r.Commits); n > 0 {
		commitInfo = fmt.Sprintf(" (%d new commit", n)
		if n != 1 {
			commitInfo += "s"
		}
		commitInfo += ")"
	}
	indicator := " "
	if len(r.Commits) > 0 {
		indicator = "▸"
	}
	return cursor + indicator + " " + th.SuccessStyle.Render("✓ "+r.Name) + th.MutedTextStyle.Render(commitInfo)
}
