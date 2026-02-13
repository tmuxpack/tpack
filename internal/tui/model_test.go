package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

func newTestModel(t *testing.T, plugins []plugin.Plugin) Model {
	t.Helper()
	cfg := &config.Config{PluginPath: t.TempDir() + "/"}
	deps := Deps{
		Cloner:    git.NewMockCloner(),
		Puller:    git.NewMockPuller(),
		Validator: git.NewMockValidator(),
		Fetcher:   git.NewMockFetcher(),
	}
	return NewModel(cfg, plugins, deps)
}

func TestNewModel_InitialState(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
		{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank"},
	}
	m := newTestModel(t, plugins)

	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
	if m.operation != OpNone {
		t.Errorf("expected OpNone, got %d", m.operation)
	}
	if len(m.plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(m.plugins))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if len(m.selected) != 0 {
		t.Errorf("expected empty selection, got %d", len(m.selected))
	}
}

func TestNewModel_PluginStatus(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}
	m := newTestModel(t, plugins)

	// No plugin dirs exist in temp dir, so all should be NotInstalled.
	for _, p := range m.plugins {
		if p.Status != StatusNotInstalled {
			t.Errorf("expected StatusNotInstalled for %s, got %s", p.Name, p.Status)
		}
	}
}

func TestUpdate_QuitKey(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
}

func TestUpdate_CursorNavigation(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "a", Spec: "user/a"},
		{Name: "b", Spec: "user/b"},
		{Name: "c", Spec: "user/c"},
	}
	m := newTestModel(t, plugins)
	m.viewHeight = 10

	// Move down.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.Update(down)
	m = result.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", m.cursor)
	}

	// Move down again.
	result, _ = m.Update(down)
	m = result.(Model)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after j, got %d", m.cursor)
	}

	// Move up.
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m.Update(up)
	m = result.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after k, got %d", m.cursor)
	}

	// Can't go above 0.
	result, _ = m.Update(up)
	m = result.(Model)
	result, _ = m.Update(up)
	m = result.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
}

func TestUpdate_WindowSize(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestView_NonEmpty(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}
	m := newTestModel(t, plugins)
	m.width = 80
	m.viewHeight = 20

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_ProgressScreen(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.totalItems = 3
	m.completedItems = 1
	m.width = 80

	view := m.View()
	if view == "" {
		t.Error("expected non-empty progress view")
	}
}

func TestStartOperation_Install(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "test-plugin", Spec: "user/test-plugin"},
	}
	m := newTestModel(t, plugins)
	// Plugin is not installed, so install should work.
	result, cmd := m.startOperation(OpInstall)
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress, got %d", m.screen)
	}
	if m.operation != OpInstall {
		t.Errorf("expected OpInstall, got %d", m.operation)
	}
	if m.totalItems != 1 {
		t.Errorf("expected 1 total item, got %d", m.totalItems)
	}
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestStartOperation_NoOps(t *testing.T) {
	// All plugins installed → install has nothing to do.
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "test", Status: StatusInstalled},
	}
	result, cmd := m.startOperation(OpInstall)
	m = result.(Model)

	if m.screen != ScreenList {
		t.Errorf("expected to stay on ScreenList, got %d", m.screen)
	}
	if cmd != nil {
		t.Error("expected nil command when no ops available")
	}
}

func TestReturnToList(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpClean
	m.processing = false

	m = m.returnToList()

	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
	if m.operation != OpNone {
		t.Errorf("expected OpNone, got %d", m.operation)
	}
}
