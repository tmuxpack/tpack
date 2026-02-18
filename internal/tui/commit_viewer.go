package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmuxpack/tpack/internal/git"
)

// commitViewerReservedLines is the overhead for title, help, and padding in the commit viewer.
const commitViewerReservedLines = 13

// maxVisible returns the number of commit rows that fit in the current height.
func (m *CommitViewer) maxVisible() int {
	v := m.height - commitViewerReservedLines
	if v < MinViewHeight {
		return MinViewHeight
	}
	return v
}

// CommitViewer is a Bubble Tea model for viewing a list of commits.
type CommitViewer struct {
	name    string
	commits []git.Commit
	theme   Theme

	scroll    scrollState
	width     int
	height    int
	sizeKnown bool
}

// NewCommitViewer creates a new CommitViewer model.
func NewCommitViewer(name string, commits []git.Commit, theme Theme) CommitViewer {
	return CommitViewer{
		name:      name,
		commits:   commits,
		theme:     theme,
		width:     FixedWidth,
		height:    FixedHeight,
		sizeKnown: true,
	}
}

// Init implements tea.Model.
func (m CommitViewer) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m CommitViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sizeKnown = true
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, SharedKeys.ForceQuit):
			return m, tea.Quit
		case key.Matches(msg, SharedKeys.Quit), msg.String() == escKeyName:
			return m, tea.Quit
		case key.Matches(msg, ListKeys.Up):
			m.scroll.moveUp()
		case key.Matches(msg, ListKeys.Down):
			m.scroll.moveDown(len(m.commits), m.maxVisible())
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m CommitViewer) View() string {
	var b strings.Builder

	// Title
	title := commitTitle(m.name, len(m.commits))
	b.WriteString(m.centerText(m.theme.TitleStyle.Render(title)))
	b.WriteString("\n\n")

	renderCommitList(&b, m.commits, m.scroll, m.maxVisible(), m.theme)

	// Help — pinned to bottom.
	help := m.centerText(m.theme.renderHelp(m.width, SharedKeys.Quit))

	return m.theme.BaseStyle.Render(padToBottom(b.String(), help, m.height))
}

// commitTitle builds the title string for the commit viewer.
func commitTitle(name string, count int) string {
	title := fmt.Sprintf("  %s — %d new commit", name, count)
	if count != 1 {
		title += "s"
	}
	title += "  "
	return title
}

// renderCommitList writes the scrollable commit list into b.
func renderCommitList(b *strings.Builder, commits []git.Commit, scroll scrollState, maxVisible int, theme Theme) {
	visible := min(len(commits), maxVisible)
	end := min(scroll.scrollOffset+visible, len(commits))

	top, bottom, dataStart, dataEnd := theme.renderScrollIndicators(scroll.scrollOffset, end, len(commits))
	b.WriteString(top)

	for i := dataStart; i < dataEnd; i++ {
		c := commits[i]
		cursor := "  "
		if i == scroll.cursor {
			cursor = "> "
		}
		b.WriteString(cursor + theme.MutedTextStyle.Render(c.Hash) + " " + c.Message + "\n")
	}

	b.WriteString(bottom)
}

// CommitViewerIdealSize returns the fixed popup dimensions for the commit viewer.
func CommitViewerIdealSize(_ string, _ []git.Commit) (width, height int) {
	return FixedWidth, FixedHeight
}

// centerText centers a single-line element within the available width.
func (m *CommitViewer) centerText(text string) string {
	if !m.sizeKnown || m.width <= 0 {
		return text
	}
	contentWidth := m.width - BaseStylePadding
	if lipgloss.Width(text) >= contentWidth {
		return text
	}
	return lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, text)
}

// RunCommitViewer launches the commit viewer TUI.
func RunCommitViewer(name string, commits []git.Commit, theme Theme) error {
	m := NewCommitViewer(name, commits, theme)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("commit viewer: %w", err)
	}
	return nil
}

// viewCommits renders the inline commit viewer screen for the main Model.
func (m *Model) viewCommits() string {
	var b strings.Builder

	// Title
	title := commitTitle(m.commitViewName, len(m.commitViewCommits))
	b.WriteString(m.centerText(m.theme.TitleStyle.Render(title)))
	b.WriteString("\n\n")

	renderCommitList(&b, m.commitViewCommits, m.commitScroll, m.commitMaxVisible(), m.theme)

	// Help — pinned to bottom.
	help := m.centerText(m.theme.renderHelp(m.width, SharedKeys.Back, SharedKeys.Quit))

	return padToBottom(b.String(), help, m.height)
}

// commitMaxVisible returns the number of commit rows that fit in the current height.
func (m *Model) commitMaxVisible() int {
	v := m.height - commitViewerReservedLines
	if v < MinViewHeight {
		return MinViewHeight
	}
	return v
}
