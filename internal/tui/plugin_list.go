package tui

import (
	"fmt"
	"strings"
)

// viewList renders the list screen.
func (m *Model) viewList() string {
	var b strings.Builder

	// Title
	b.WriteString(m.centerText(m.theme.TitleStyle.Render("  TPM Plugin Manager  ")))
	b.WriteString("\n")

	subtitle := m.statusSummary()
	b.WriteString(m.centerText(m.theme.SubtitleStyle.Render(subtitle)))
	b.WriteString("\n")

	if len(m.plugins) == 0 {
		b.WriteString(m.centerText(m.theme.MutedTextStyle.Render("  No plugins configured in tmux.conf")))
		b.WriteString("\n")
	} else {
		// Build table as a block (header + separator + rows + scroll indicators)
		// so internal column alignment is preserved when centering.
		var tb strings.Builder

		// Table header
		nameCol := "name"
		statusCol := "status"
		header := fmt.Sprintf("  %-*s  %s", m.nameColWidth(), nameCol, statusCol)
		tb.WriteString(m.theme.MutedTextStyle.Render(header))
		tb.WriteString("\n")
		tb.WriteString(m.theme.MutedTextStyle.Render("  " + strings.Repeat("─", m.tableWidth())))
		tb.WriteString("\n")

		// Calculate visible range
		viewHeight := m.viewHeight
		if viewHeight <= 0 {
			viewHeight = len(m.plugins)
		}
		start, end := calculateVisibleRange(m.listScroll.scrollOffset, viewHeight, len(m.plugins))

		// Scroll indicators (indicators replace data rows to keep layout stable)
		topIndicator, bottomIndicator, dataStart, dataEnd := m.theme.renderScrollIndicators(start, end, len(m.plugins))
		tb.WriteString(topIndicator)

		// Plugin rows
		for i := dataStart; i < dataEnd; i++ {
			p := m.plugins[i]
			cursor := renderCursor(i == m.listScroll.cursor)

			var checkbox string
			if m.multiSelectActive {
				checkbox = m.theme.renderCheckbox(m.selected[i]) + " "
			}

			status := m.renderStatus(p.Status)

			row := fmt.Sprintf("%s%s%-*s  %s", cursor, checkbox, m.nameColWidth(), p.Name, status)

			if i == m.listScroll.cursor {
				row = m.theme.SelectedRowStyle.Render(row)
			} else if m.selected[i] {
				row = m.theme.CheckedStyle.Render(row)
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
		b.WriteString(m.centerText(m.theme.OrphanStyle.Render("Orphaned: " + strings.Join(names, ", "))))
		b.WriteString("\n")
	}

	// Help bar — context-aware actions, pinned to bottom.
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
	help := m.centerText(m.theme.renderHelp(m.width, helpPairs...))

	return padToBottom(b.String(), help, m.height)
}

// targetHasStatus checks the statuses of the target plugins (selected or cursor).
func (m *Model) targetHasStatus() (hasNotInstalled, hasInstalled bool) {
	indices := m.targetIndices()
	for _, i := range indices {
		if m.plugins[i].Status.IsInstalled() {
			hasInstalled = true
		} else {
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

// statusSummary returns the subtitle text with plugin counts.
func (m *Model) statusSummary() string {
	installed := 0
	notInstalled := 0
	outdated := 0
	for _, p := range m.plugins {
		switch p.Status {
		case StatusInstalled, StatusChecking, StatusCheckFailed:
			installed++
		case StatusNotInstalled:
			notInstalled++
		case StatusOutdated:
			outdated++
		}
	}
	s := fmt.Sprintf("%d installed, %d not installed", installed, notInstalled)
	if outdated > 0 {
		s += fmt.Sprintf(", %d outdated", outdated)
	}
	return s
}

// renderStatus returns the styled status text for a plugin.
func (m *Model) renderStatus(s PluginStatus) string {
	switch s {
	case StatusInstalled:
		return m.theme.StatusInstalledStyle.Render("Installed")
	case StatusNotInstalled:
		return m.theme.StatusNotInstalledStyle.Render("Not Installed")
	case StatusChecking:
		return m.theme.StatusInstalledStyle.Render("Installed") + " " + m.checkSpinner.View()
	case StatusOutdated:
		return m.theme.StatusOutdatedStyle.Render("Outdated")
	case StatusCheckFailed:
		return m.theme.StatusInstalledStyle.Render("Installed") + " " + m.theme.StatusCheckFailedStyle.Render("⚠")
	default:
		return ""
	}
}
