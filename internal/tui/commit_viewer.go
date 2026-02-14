package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/git"
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

	scroll    scrollState
	width     int
	height    int
	sizeKnown bool
}

// NewCommitViewer creates a new CommitViewer model.
func NewCommitViewer(name string, commits []git.Commit) CommitViewer {
	return CommitViewer{
		name:      name,
		commits:   commits,
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
	title := fmt.Sprintf("  %s — %d new commit", m.name, len(m.commits))
	if len(m.commits) != 1 {
		title += "s"
	}
	title += "  "
	b.WriteString(m.centerText(TitleStyle.Render(title)))
	b.WriteString("\n\n")

	// Commit list with scroll indicators
	visible := m.maxVisible()
	if len(m.commits) < visible {
		visible = len(m.commits)
	}
	end := m.scroll.scrollOffset + visible
	if end > len(m.commits) {
		end = len(m.commits)
	}

	top, bottom, dataStart, dataEnd := renderScrollIndicators(m.scroll.scrollOffset, end, len(m.commits))
	b.WriteString(top)

	for i := dataStart; i < dataEnd; i++ {
		c := m.commits[i]
		cursor := "  "
		if i == m.scroll.cursor {
			cursor = "> "
		}
		b.WriteString(cursor + MutedTextStyle.Render(c.Hash) + " " + c.Message + "\n")
	}

	b.WriteString(bottom)

	// Help — pinned to bottom.
	help := m.centerText(renderHelp(m.width, "q", "quit"))

	return BaseStyle.Render(padToBottom(b.String(), help, m.height))
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
func RunCommitViewer(name string, commits []git.Commit) error {
	m := NewCommitViewer(name, commits)
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
	title := fmt.Sprintf("  %s — %d new commit", m.commitViewName, len(m.commitViewCommits))
	if len(m.commitViewCommits) != 1 {
		title += "s"
	}
	title += "  "
	b.WriteString(m.centerText(TitleStyle.Render(title)))
	b.WriteString("\n\n")

	// Commit list with scroll indicators
	visible := m.commitMaxVisible()
	if len(m.commitViewCommits) < visible {
		visible = len(m.commitViewCommits)
	}
	end := m.commitScroll.scrollOffset + visible
	if end > len(m.commitViewCommits) {
		end = len(m.commitViewCommits)
	}

	top, bottom, dataStart, dataEnd := renderScrollIndicators(m.commitScroll.scrollOffset, end, len(m.commitViewCommits))
	b.WriteString(top)

	for i := dataStart; i < dataEnd; i++ {
		c := m.commitViewCommits[i]
		cursor := "  "
		if i == m.commitScroll.cursor {
			cursor = "> "
		}
		b.WriteString(cursor + MutedTextStyle.Render(c.Hash) + " " + c.Message + "\n")
	}

	b.WriteString(bottom)

	// Help — pinned to bottom.
	help := m.centerText(renderHelp(m.width, "esc", "back", "q", "quit"))

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
