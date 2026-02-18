package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	b.WriteString("\n")

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
	stats := m.renderStats()
	b.WriteString(m.centerText(m.theme.MutedTextStyle.Render(stats)))

	// Show results detail if complete
	visible := m.displayResults()
	if !m.processing && len(m.results) > 0 {
		if len(visible) > 0 {
			b.WriteString("\n\n")
			b.WriteString(m.centerBlock(m.renderResults()))
		}

		var bindings []key.Binding
		if len(visible) > 0 {
			r := visible[m.resultScroll.cursor]
			if r.Success && len(r.Commits) > 0 {
				bindings = append(bindings, ProgressKeys.ViewCommits)
			}
		}
		if m.autoOp != OpNone {
			bindings = append(bindings, SharedKeys.Quit)
		} else {
			bindings = append(bindings, SharedKeys.Quit, ProgressKeys.BackToList)
		}
		help := m.centerText(m.theme.renderHelp(m.width, bindings...))
		return padToBottom(b.String(), help, m.height)
	}

	return b.String()
}

// displayResults returns the results to show in the list. For updates, only
// plugins that actually changed or failed are shown; already up-to-date
// plugins are excluded. For other operations, all results are returned.
func (m *Model) displayResults() []ResultItem {
	if m.operation != OpUpdate {
		return m.results
	}
	var filtered []ResultItem
	for _, r := range m.results {
		if !r.Success || len(r.Commits) > 0 {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// renderResults renders the completed results list with cursor, commit counts, and scrolling.
func (m *Model) renderResults() string {
	visible := m.displayResults()
	viewHeight := m.resultMaxVisible()
	if viewHeight <= 0 {
		viewHeight = len(visible)
	}
	start, end := calculateVisibleRange(m.resultScroll.scrollOffset, viewHeight, len(visible))
	topIndicator, bottomIndicator, dataStart, dataEnd := m.theme.renderScrollIndicators(start, end, len(visible))

	var rb strings.Builder
	rb.WriteString(topIndicator)
	for i := dataStart; i < dataEnd; i++ {
		r := visible[i]
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

// renderStats returns the summary stats string for the progress screen.
func (m *Model) renderStats() string {
	if m.operation == OpUpdate {
		return m.renderUpdateStats()
	}
	successCount := 0
	failCount := 0
	for _, r := range m.results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	return fmt.Sprintf("%s %d successful  %s %d failed",
		m.theme.SuccessStyle.Render("✓"), successCount,
		m.theme.ErrorStyle.Render("✗"), failCount,
	)
}

// renderUpdateStats returns the categorized summary for update operations.
func (m *Model) renderUpdateStats() string {
	upToDate := 0
	updated := 0
	failed := 0
	for _, r := range m.results {
		switch {
		case !r.Success:
			failed++
		case len(r.Commits) > 0:
			updated++
		default:
			upToDate++
		}
	}
	return fmt.Sprintf("%d already up-to-date, %s %d updated, %s %d failed",
		upToDate,
		m.theme.SuccessStyle.Render("✓"), updated,
		m.theme.ErrorStyle.Render("✗"), failed,
	)
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
