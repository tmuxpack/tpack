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
		var rb strings.Builder
		for _, r := range m.results {
			if r.Success {
				rb.WriteString(SuccessStyle.Render("✓ " + r.Name))
			} else {
				rb.WriteString(ErrorStyle.Render("✗ " + r.Name + ": " + r.Message))
			}
			rb.WriteString("\n")
		}
		b.WriteString(m.centerBlock(strings.TrimRight(rb.String(), "\n")))
		b.WriteString("\n\n")
		b.WriteString(m.centerText(renderHelp(m.width, "q", "quit", "esc", "back to list")))
	}

	return b.String()
}
