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
	b.WriteString(m.centerText(TitleStyle.Render(title)))
	b.WriteString("\n")

	// Counter
	counter := fmt.Sprintf("Processing %d of %d plugins", m.completedItems, m.totalItems)
	b.WriteString(m.centerText(SubtitleStyle.Render(counter)))
	b.WriteString("\n\n")

	// Current item
	if m.currentItemName != "" {
		b.WriteString(m.centerText("Current: " + m.currentItemName))
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
		SuccessStyle.Render("✓"), successCount,
		ErrorStyle.Render("✗"), failCount,
	)
	b.WriteString(m.centerText(MutedTextStyle.Render(stats)))

	// Show results detail if complete
	if !m.processing && len(m.results) > 0 {
		b.WriteString("\n\n")
		b.WriteString(m.centerBlock(m.renderResults()))
		b.WriteString("\n\n")
		helpKeys := []string{"enter", "view commits"}
		if m.autoOp != OpNone {
			helpKeys = append(helpKeys, "q", "quit")
		} else {
			helpKeys = append(helpKeys, "q", "quit", "esc", "back to list")
		}
		b.WriteString(m.centerText(renderHelp(m.width, helpKeys...)))
	}

	return b.String()
}

// renderResults renders the completed results list with cursor and commit counts.
func (m *Model) renderResults() string {
	var rb strings.Builder
	for i, r := range m.results {
		cursor := "  "
		if i == m.resultCursor {
			cursor = "> "
		}
		if r.Success {
			rb.WriteString(renderSuccessResult(cursor, r))
		} else {
			rb.WriteString(cursor + "  " + ErrorStyle.Render("✗ "+r.Name+": "+r.Message))
		}
		rb.WriteString("\n")
	}
	return strings.TrimRight(rb.String(), "\n")
}

// renderSuccessResult renders a single successful result line with commit count and indicator.
func renderSuccessResult(cursor string, r ResultItem) string {
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
	return cursor + indicator + " " + SuccessStyle.Render("✓ "+r.Name) + MutedTextStyle.Render(commitInfo)
}
