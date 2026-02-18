package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmuxpack/tpack/internal/registry"
)

func newBrowseModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t, nil)
	m.screen = ScreenBrowse
	m.browseRegistry = &registry.Registry{
		Categories: []string{"theme", "session", "utility"},
		Plugins: []registry.RegistryItem{
			{Repo: "catppuccin/tmux", Description: "Pastel theme", Category: "theme", Stars: 1250},
			{Repo: "tmux-plugins/tmux-resurrect", Description: "Persist environment", Category: "session", Stars: 11400},
			{Repo: "tmux-plugins/tmux-sensible", Description: "Sensible defaults", Category: "utility", Stars: 5000},
		},
	}
	m.browseResults = m.browseRegistry.Plugins
	m.browseCategory = -1
	return m
}

func TestBrowseUpdate_TabCyclesCategory(t *testing.T) {
	m := newBrowseModel(t)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := m.Update(msg)
	m = result.(Model)
	if m.browseCategory != 0 {
		t.Errorf("expected category 0, got %d", m.browseCategory)
	}

	result, _ = m.Update(msg)
	m = result.(Model)
	if m.browseCategory != 1 {
		t.Errorf("expected category 1, got %d", m.browseCategory)
	}

	result, _ = m.Update(msg)
	m = result.(Model)
	result, _ = m.Update(msg)
	m = result.(Model)
	if m.browseCategory != -1 {
		t.Errorf("expected category -1 (all), got %d", m.browseCategory)
	}
}

func TestBrowseUpdate_CursorNavigation(t *testing.T) {
	m := newBrowseModel(t)

	down := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.Update(down)
	m = result.(Model)
	if m.browseScroll.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.browseScroll.cursor)
	}

	up := tea.KeyMsg{Type: tea.KeyUp}
	result, _ = m.Update(up)
	m = result.(Model)
	if m.browseScroll.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.browseScroll.cursor)
	}
}

func TestBrowseUpdate_EscReturnsToList(t *testing.T) {
	m := newBrowseModel(t)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := m.Update(msg)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestUpdate_BrowseKeyOpensBrowse(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenBrowse {
		t.Errorf("expected ScreenBrowse, got %d", m.screen)
	}
	if cmd == nil {
		t.Error("expected non-nil command (registry fetch)")
	}
}

func TestBrowseRoundTrip(t *testing.T) {
	m := newTestModel(t, nil)

	browse := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	result, _ := m.Update(browse)
	m = result.(Model)
	if m.screen != ScreenBrowse {
		t.Fatalf("expected ScreenBrowse, got %d", m.screen)
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
	if m.browseLoading {
		t.Error("expected loading to be false after fetch")
	}
	if len(m.browseResults) != 1 {
		t.Errorf("expected 1 result, got %d", len(m.browseResults))
	}

	esc := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ = m.Update(esc)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestInstallFromBrowse_AddsToPluginsAndStartsInstall(t *testing.T) {
	m := newBrowseModel(t)
	m.cfg.TmuxConf = filepath.Join(t.TempDir(), "tmux.conf")
	os.WriteFile(m.cfg.TmuxConf, []byte("# tmux config\n"), 0o644)

	m.browseScroll.cursor = 0

	result, cmd := m.installFromBrowse()
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
