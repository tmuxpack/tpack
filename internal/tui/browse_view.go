package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

func (m *Model) viewBrowse() string {
	var b strings.Builder

	b.WriteString(m.centerText(m.theme.TitleStyle.Render("  Browse Plugins  ")))
	b.WriteString("\n")

	b.WriteString(m.centerText(m.renderCategoryBar()))
	b.WriteString("\n\n")

	if m.searching || m.browseQuery != "" {
		b.WriteString("  " + m.browseInput.View())
	}

	b.WriteString("\n")

	switch {
	case m.browseLoading:
		b.WriteString(m.centerText(m.checkSpinner.View() + " Loading registry..."))
		b.WriteString("\n")
	case m.browseErr != nil:
		b.WriteString(m.centerText(m.theme.ErrorStyle.Render("Error: " + m.browseErr.Error())))
		b.WriteString("\n")
	case len(m.browseResults) == 0:
		b.WriteString(m.centerText(m.theme.MutedTextStyle.Render("  No plugins found")))
		b.WriteString("\n")
	default:
		b.WriteString(m.renderBrowseResults())
	}

	var bindings []key.Binding

	if m.searching {
		bindings = []key.Binding{BrowseKeys.Apply, BrowseKeys.Cancel}
	} else {
		bindings = []key.Binding{BrowseKeys.Open, BrowseKeys.Filter, BrowseKeys.Category, ListKeys.Install, SharedKeys.Back}
	}

	status := ""
	if m.browseStatus != "" {
		status = m.centerText(m.theme.MutedTextStyle.Render(m.browseStatus))
	}
	help := m.centerText(m.theme.renderHelp(m.width, bindings...))
	return padToBottom(b.String(), status+"\n"+help, m.height)
}

func (m *Model) renderCategoryBar() string {
	if m.browseRegistry == nil {
		return ""
	}

	var parts []string
	if m.browseCategory == -1 {
		parts = append(parts, m.theme.BrowseCategoryStyle.Render("[all]"))
	} else {
		parts = append(parts, m.theme.MutedTextStyle.Render("all"))
	}

	for i, cat := range m.browseRegistry.Categories {
		if i == m.browseCategory {
			parts = append(parts, m.theme.BrowseCategoryStyle.Render("["+cat+"]"))
		} else {
			parts = append(parts, m.theme.MutedTextStyle.Render(cat))
		}
	}
	return strings.Join(parts, "  ")
}

func (m *Model) renderBrowseResults() string {
	var tb strings.Builder

	viewHeight := m.browseViewHeight()
	start, end := calculateVisibleRange(m.browseScroll.scrollOffset, viewHeight, len(m.browseResults))
	topIndicator, bottomIndicator, dataStart, dataEnd := m.theme.renderScrollIndicators(start, end, len(m.browseResults))
	tb.WriteString(topIndicator)

	for i := dataStart; i < dataEnd; i++ {
		p := m.browseResults[i]
		cursor := renderCursor(i == m.browseScroll.cursor)

		stars := m.theme.BrowseStarsStyle.Render(formatStars(p.Stars))
		repo := m.theme.BrowseRepoStyle.Render(p.Repo)

		installed := ""
		for _, pl := range m.plugins {
			if pl.Spec == p.Repo || pl.Name == pluginNameFromRepo(p.Repo) {
				installed = " " + m.theme.BrowseInstalledStyle.Render("(installed)")
				break
			}
		}

		row := fmt.Sprintf("%s%s  %s%s", cursor, stars, repo, installed)
		if i == m.browseScroll.cursor {
			row = m.theme.SelectedRowStyle.Render(row)
		}
		tb.WriteString(row)
		tb.WriteString("\n")

		desc := fmt.Sprintf("       %s", m.theme.BrowseDescStyle.Render(p.Description))
		tb.WriteString(desc)
		tb.WriteString("\n")
	}

	tb.WriteString(bottomIndicator)
	return strings.TrimRight(tb.String(), "\n")
}

func (m *Model) browseViewHeight() int {
	// TODO: make this dynamic based on the actual layout and content above/below the list
	reserved := 10
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
