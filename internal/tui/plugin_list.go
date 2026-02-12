package tui

import (
	"fmt"
	"strings"
)

// viewList renders the list screen.
func (m *Model) viewList() string {
	var b strings.Builder

	// Title
	b.WriteString(m.centerText(TitleStyle.Render("  TPM Plugin Manager  ")))
	b.WriteString("\n")

	// Subtitle: counts
	installed := 0
	notInstalled := 0
	for _, p := range m.plugins {
		if p.Status == StatusInstalled {
			installed++
		} else {
			notInstalled++
		}
	}
	subtitle := fmt.Sprintf("%d installed, %d not installed", installed, notInstalled)
	b.WriteString(m.centerText(SubtitleStyle.Render(subtitle)))
	b.WriteString("\n")

	if len(m.plugins) == 0 {
		b.WriteString(m.centerText(MutedTextStyle.Render("  No plugins configured in tmux.conf")))
		b.WriteString("\n")
	} else {
		// Build table as a block (header + separator + rows + scroll indicators)
		// so internal column alignment is preserved when centering.
		var tb strings.Builder

		// Table header
		nameCol := "name"
		statusCol := "status"
		header := fmt.Sprintf("  %-*s  %s", m.nameColWidth(), nameCol, statusCol)
		tb.WriteString(MutedTextStyle.Render(header))
		tb.WriteString("\n")
		tb.WriteString(MutedTextStyle.Render("  " + strings.Repeat("─", m.tableWidth())))
		tb.WriteString("\n")

		// Calculate visible range
		viewHeight := m.viewHeight
		if viewHeight <= 0 {
			viewHeight = len(m.plugins)
		}
		start, end := calculateVisibleRange(m.scrollOffset, viewHeight, len(m.plugins))

		// Scroll indicators
		topIndicator, bottomIndicator := renderScrollIndicators(start, end, len(m.plugins))
		tb.WriteString(topIndicator)

		// Plugin rows
		for i := start; i < end; i++ {
			p := m.plugins[i]
			cursor := renderCursor(i == m.cursor)

			var checkbox string
			if m.multiSelectActive {
				checkbox = renderCheckbox(m.selected[i]) + " "
			}

			// Status badge
			var status string
			if p.Status == StatusInstalled {
				status = StatusInstalledStyle.Render("Installed")
			} else {
				status = StatusNotInstalledStyle.Render("Not Installed")
			}

			row := fmt.Sprintf("%s%s%-*s  %s", cursor, checkbox, m.nameColWidth(), p.Name, status)

			if i == m.cursor {
				row = SelectedRowStyle.Render(row)
			} else if m.selected[i] {
				row = CheckedStyle.Render(row)
			}

			tb.WriteString(row)
			tb.WriteString("\n")
		}

		tb.WriteString(bottomIndicator)

		// Center the table block while preserving column alignment.
		b.WriteString(m.centerBlock(strings.TrimRight(tb.String(), "\n")))
		b.WriteString("\n")
	}

	// Orphans section
	if len(m.orphans) > 0 {
		b.WriteString("\n")
		names := make([]string, len(m.orphans))
		for i, o := range m.orphans {
			names[i] = o.Name
		}
		b.WriteString(m.centerText(OrphanStyle.Render("Orphaned: " + strings.Join(names, ", "))))
		b.WriteString("\n")
	}

	// Help bar — context-aware actions only.
	b.WriteString("\n")
	var helpPairs []string
	hasNotInstalled, hasInstalled := m.targetHasStatus()
	if hasNotInstalled {
		helpPairs = append(helpPairs, "i", "install")
	}
	if hasInstalled {
		helpPairs = append(helpPairs, "u", "update", "x", "uninstall")
	}
	if len(m.orphans) > 0 {
		helpPairs = append(helpPairs, "c", "clean")
	}
	helpPairs = append(helpPairs, "q", "quit")
	b.WriteString(m.centerText(renderHelp(m.width, helpPairs...)))

	return b.String()
}

// targetHasStatus checks the statuses of the target plugins (selected or cursor).
func (m *Model) targetHasStatus() (hasNotInstalled, hasInstalled bool) {
	indices := m.targetIndices()
	for _, i := range indices {
		switch m.plugins[i].Status {
		case StatusInstalled:
			hasInstalled = true
		case StatusNotInstalled:
			hasNotInstalled = true
		}
		if hasInstalled && hasNotInstalled {
			return
		}
	}
	return
}

// nameColWidth returns the width of the name column.
func (m *Model) nameColWidth() int {
	maxLen := 10
	for _, p := range m.plugins {
		if len(p.Name) > maxLen {
			maxLen = len(p.Name)
		}
	}
	return maxLen + 2
}

// tableWidth returns the total table width.
func (m *Model) tableWidth() int {
	return m.nameColWidth() + 2 + 14 // status col ~14 chars
}

// calculateVisibleRange returns the start and end indices for visible items.
func calculateVisibleRange(offset, viewHeight, total int) (int, int) {
	start := offset
	end := offset + viewHeight
	if end > total {
		end = total
	}
	return start, end
}
