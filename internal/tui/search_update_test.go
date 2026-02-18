package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmuxpack/tpack/internal/registry"
)

func newSearchModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t, nil)
	m.screen = ScreenSearch
	m.searchRegistry = &registry.Registry{
		Categories: []string{"theme", "session", "utility"},
		Plugins: []registry.RegistryItem{
			{Repo: "catppuccin/tmux", Description: "Pastel theme", Category: "theme", Stars: 1250},
			{Repo: "tmux-plugins/tmux-resurrect", Description: "Persist environment", Category: "session", Stars: 11400},
			{Repo: "tmux-plugins/tmux-sensible", Description: "Sensible defaults", Category: "utility", Stars: 5000},
		},
	}
	m.searchResults = m.searchRegistry.Plugins
	m.searchCategory = -1
	return m
}

func TestSearchUpdate_TabCyclesCategory(t *testing.T) {
	m := newSearchModel(t)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := m.Update(msg)
	m = result.(Model)
	if m.searchCategory != 0 {
		t.Errorf("expected category 0, got %d", m.searchCategory)
	}

	result, _ = m.Update(msg)
	m = result.(Model)
	if m.searchCategory != 1 {
		t.Errorf("expected category 1, got %d", m.searchCategory)
	}

	result, _ = m.Update(msg)
	m = result.(Model)
	result, _ = m.Update(msg)
	m = result.(Model)
	if m.searchCategory != -1 {
		t.Errorf("expected category -1 (all), got %d", m.searchCategory)
	}
}

func TestSearchUpdate_CursorNavigation(t *testing.T) {
	m := newSearchModel(t)

	down := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.Update(down)
	m = result.(Model)
	if m.searchScroll.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.searchScroll.cursor)
	}

	up := tea.KeyMsg{Type: tea.KeyUp}
	result, _ = m.Update(up)
	m = result.(Model)
	if m.searchScroll.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.searchScroll.cursor)
	}
}

func TestSearchUpdate_EscReturnsToList(t *testing.T) {
	m := newSearchModel(t)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := m.Update(msg)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestUpdate_SearchKeyOpensSearch(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenSearch {
		t.Errorf("expected ScreenSearch, got %d", m.screen)
	}
	if cmd == nil {
		t.Error("expected non-nil command (registry fetch)")
	}
}

func TestSearchRoundTrip(t *testing.T) {
	m := newTestModel(t, nil)

	browse := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	result, _ := m.Update(browse)
	m = result.(Model)
	if m.screen != ScreenSearch {
		t.Fatalf("expected ScreenSearch, got %d", m.screen)
	}

	result, _ = m.Update(registryFetchResultMsg{
		Registry: &registry.Registry{
			Categories: []string{"theme"},
			Plugins: []registry.RegistryItem{
				{Repo: "catppuccin/tmux", Description: "Theme", Category: "theme", Stars: 100},
			},
		},
	})
	m = result.(Model)
	if m.searchLoading {
		t.Error("expected loading to be false after fetch")
	}
	if len(m.searchResults) != 1 {
		t.Errorf("expected 1 result, got %d", len(m.searchResults))
	}

	esc := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ = m.Update(esc)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestInstallFromSearch_AddsToPluginsAndStartsInstall(t *testing.T) {
	m := newSearchModel(t)
	m.cfg.TmuxConf = filepath.Join(t.TempDir(), "tmux.conf")
	os.WriteFile(m.cfg.TmuxConf, []byte("# tmux config\n"), 0o644)

	m.searchScroll.cursor = 0

	result, cmd := m.installFromSearch()
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress, got %d", m.screen)
	}
	if m.totalItems != 1 {
		t.Errorf("expected 1 pending install, got %d", m.totalItems)
	}
	if cmd == nil {
		t.Error("expected non-nil install command")
	}

	data, _ := os.ReadFile(m.cfg.TmuxConf)
	if !strings.Contains(string(data), "catppuccin/tmux") {
		t.Error("expected plugin line in tmux.conf")
	}
}
