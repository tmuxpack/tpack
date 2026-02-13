package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/git"
)

func testCommits() []git.Commit {
	return []git.Commit{
		{Hash: "abc1234", Message: "add feature X"},
		{Hash: "def5678", Message: "fix bug Y"},
		{Hash: "ghi9012", Message: "refactor Z"},
	}
}

func TestCommitViewer_View_ShowsTitle(t *testing.T) {
	m := NewCommitViewer("tmux-sensible", testCommits())

	view := m.View()
	if !strings.Contains(view, "tmux-sensible") {
		t.Error("expected view to contain plugin name")
	}
	if !strings.Contains(view, "3 new commits") {
		t.Error("expected view to show '3 new commits'")
	}
}

func TestCommitViewer_View_SingleCommit(t *testing.T) {
	commits := []git.Commit{{Hash: "abc1234", Message: "single change"}}
	m := NewCommitViewer("test-plugin", commits)

	view := m.View()
	if !strings.Contains(view, "1 new commit") {
		t.Error("expected view to show '1 new commit' (singular)")
	}
	if strings.Contains(view, "1 new commits") {
		t.Error("expected singular 'commit', got plural")
	}
}

func TestCommitViewer_View_ShowsCommitHashes(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	view := m.View()
	for _, c := range testCommits() {
		if !strings.Contains(view, c.Hash) {
			t.Errorf("expected view to contain hash %q", c.Hash)
		}
		if !strings.Contains(view, c.Message) {
			t.Errorf("expected view to contain message %q", c.Message)
		}
	}
}

func TestCommitViewer_View_ShowsHelp(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	view := m.View()
	if !strings.Contains(view, "quit") {
		t.Error("expected view to contain 'quit' in help")
	}
}

func TestCommitViewer_Navigation(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	if m.cursor != 0 {
		t.Errorf("expected initial cursor at 0, got %d", m.cursor)
	}

	// Move down.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.Update(down)
	m = result.(CommitViewer)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", m.cursor)
	}

	// Move down again.
	result, _ = m.Update(down)
	m = result.(CommitViewer)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after j, got %d", m.cursor)
	}

	// Can't go past last item.
	result, _ = m.Update(down)
	m = result.(CommitViewer)
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Move up.
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m.Update(up)
	m = result.(CommitViewer)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after k, got %d", m.cursor)
	}

	// Move up to 0.
	result, _ = m.Update(up)
	m = result.(CommitViewer)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	// Can't go above 0.
	result, _ = m.Update(up)
	m = result.(CommitViewer)
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestCommitViewer_QuitOnQ(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command on q")
	}
}

func TestCommitViewer_QuitOnEsc(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command on Esc")
	}
}

func TestCommitViewer_WindowSize(t *testing.T) {
	m := NewCommitViewer("test", testCommits())

	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = result.(CommitViewer)

	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
	if m.height != 30 {
		t.Errorf("expected height 30, got %d", m.height)
	}
	if !m.sizeKnown {
		t.Error("expected sizeKnown to be true")
	}
}

func TestCommitViewer_ScrollIndicators(t *testing.T) {
	// Create more commits than maxVisible.
	m := NewCommitViewer("test", nil)
	maxVis := m.maxVisible()
	commits := make([]git.Commit, maxVis+5)
	for i := range commits {
		commits[i] = git.Commit{Hash: "abc1234", Message: "commit"}
	}

	m = NewCommitViewer("test", commits)

	view := m.View()
	// Should show "more below" but not "more above" at start.
	if !strings.Contains(view, "more below") {
		t.Error("expected 'more below' indicator when commits exceed visible limit")
	}
	if strings.Contains(view, "more above") {
		t.Error("expected no 'more above' indicator at start")
	}

	// Scroll to bottom.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	var result tea.Model = m
	for i := 0; i < len(commits)-1; i++ {
		result, _ = result.(CommitViewer).Update(down)
	}
	m = result.(CommitViewer)

	view = m.View()
	if !strings.Contains(view, "more above") {
		t.Error("expected 'more above' indicator after scrolling down")
	}
}

func TestCommitViewerIdealSize(t *testing.T) {
	commits := testCommits()
	w, h := CommitViewerIdealSize("test-plugin", commits)

	if w <= 0 {
		t.Errorf("expected positive width, got %d", w)
	}
	if h <= 0 {
		t.Errorf("expected positive height, got %d", h)
	}
	// Should be wide enough to contain the longest commit line.
	if w < 20 {
		t.Errorf("expected width >= 20, got %d", w)
	}
}

func TestCommitViewer_Init(t *testing.T) {
	m := NewCommitViewer("test", testCommits())
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}
