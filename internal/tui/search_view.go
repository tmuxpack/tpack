package tui

import (
	"fmt"
	"strings"
)

func (m *Model) viewSearch() string {
	var b strings.Builder

	b.WriteString(m.centerText(m.theme.TitleStyle.Render("  Search Plugins  ")))
	b.WriteString("\n")

	b.WriteString(m.centerText(m.renderCategoryBar()))
	b.WriteString("\n")

	b.WriteString(m.centerText(m.searchInput.View()))
	b.WriteString("\n\n")

	switch {
	case m.searchLoading:
		b.WriteString(m.centerText(m.checkSpinner.View() + " Loading registry..."))
		b.WriteString("\n")
	case m.searchErr != nil:
		b.WriteString(m.centerText(m.theme.ErrorStyle.Render("Error: " + m.searchErr.Error())))
		b.WriteString("\n")
	case len(m.searchResults) == 0:
		b.WriteString(m.centerText(m.theme.MutedTextStyle.Render("  No plugins found")))
		b.WriteString("\n")
	default:
		b.WriteString(m.renderSearchResults())
	}

	helpPairs := []string{"/", "filter", "tab", "category", "enter", "install", "esc", "back"}
	help := m.centerText(m.theme.renderHelp(m.width, helpPairs...))
	return padToBottom(b.String(), help, m.height)
}

func (m *Model) renderCategoryBar() string {
	if m.searchRegistry == nil {
		return ""
	}

	var parts []string
	if m.searchCategory == -1 {
		parts = append(parts, m.theme.SearchCategoryStyle.Render("[all]"))
	} else {
		parts = append(parts, m.theme.MutedTextStyle.Render("all"))
	}

	for i, cat := range m.searchRegistry.Categories {
		if i == m.searchCategory {
			parts = append(parts, m.theme.SearchCategoryStyle.Render("["+cat+"]"))
		} else {
			parts = append(parts, m.theme.MutedTextStyle.Render(cat))
		}
	}
	return strings.Join(parts, "  ")
}

func (m *Model) renderSearchResults() string {
	var tb strings.Builder

	viewHeight := m.searchViewHeight()
	start, end := calculateVisibleRange(m.searchScroll.scrollOffset, viewHeight, len(m.searchResults))
	topIndicator, bottomIndicator, dataStart, dataEnd := m.theme.renderScrollIndicators(start, end, len(m.searchResults))
	tb.WriteString(topIndicator)

	for i := dataStart; i < dataEnd; i++ {
		p := m.searchResults[i]
		cursor := renderCursor(i == m.searchScroll.cursor)

		stars := m.theme.SearchStarsStyle.Render(formatStars(p.Stars))
		repo := m.theme.SearchRepoStyle.Render(p.Repo)

		installed := ""
		for _, pl := range m.plugins {
			if pl.Spec == p.Repo || pl.Name == pluginNameFromRepo(p.Repo) {
				installed = " " + m.theme.SearchInstalledStyle.Render("(installed)")
				break
			}
		}

		row := fmt.Sprintf("%s%s  %s%s", cursor, stars, repo, installed)
		if i == m.searchScroll.cursor {
			row = m.theme.SelectedRowStyle.Render(row)
		}
		tb.WriteString(row)
		tb.WriteString("\n")

		desc := fmt.Sprintf("       %s", m.theme.SearchDescStyle.Render(p.Description))
		tb.WriteString(desc)
		tb.WriteString("\n")
	}

	tb.WriteString(bottomIndicator)
	return m.centerBlock(strings.TrimRight(tb.String(), "\n"))
}

func (m *Model) searchViewHeight() int {
	reserved := 14
	available := m.height - reserved
	items := available / 2
	if items < MinViewHeight {
		return MinViewHeight
	}
	return items
}

func formatStars(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%4.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%5d", n)
}

func pluginNameFromRepo(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return repo
}
