package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

	msg := tea.KeyPressMsg{Code: tea.KeyTab}
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

	down := tea.KeyPressMsg{Code: tea.KeyDown}
	result, _ := m.Update(down)
	m = result.(Model)
	if m.browseScroll.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.browseScroll.cursor)
	}

	up := tea.KeyPressMsg{Code: tea.KeyUp}
	result, _ = m.Update(up)
	m = result.(Model)
	if m.browseScroll.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.browseScroll.cursor)
	}
}

func TestBrowseUpdate_EscReturnsToList(t *testing.T) {
	m := newBrowseModel(t)

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	result, _ := m.Update(msg)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestUpdate_BrowseKeyOpensBrowse(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.KeyPressMsg{Code: 'b', Text: "b"}
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

	browse := tea.KeyPressMsg{Code: 'b', Text: "b"}
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

	esc := tea.KeyPressMsg{Code: tea.KeyEscape}
	result, _ = m.Update(esc)
	m = result.(Model)
	if m.screen != ScreenList {
		t.Errorf("expected ScreenList, got %d", m.screen)
	}
}

func TestOpenFromBrowse_ReturnsCmd(t *testing.T) {
	m := newBrowseModel(t)
	m.browseScroll.cursor = 0

	_, cmd := m.openFromBrowse()
	if cmd == nil {
		t.Error("expected non-nil command for valid cursor")
	}
}

func TestOpenFromBrowse_InvalidCursor(t *testing.T) {
	m := newBrowseModel(t)
	m.browseScroll.cursor = 99

	_, cmd := m.openFromBrowse()
	if cmd != nil {
		t.Error("expected nil command for out-of-bounds cursor")
	}
}

func TestOpenFromBrowse_EnterKeyInNavMode(t *testing.T) {
	m := newBrowseModel(t)
	m.browseScroll.cursor = 0

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected non-nil command from Enter in navigation mode")
	}
}

func TestOpenFromBrowse_GitHubURL(t *testing.T) {
	m := newBrowseModel(t)
	m.browseScroll.cursor = 0

	result, _ := m.openFromBrowse()
	m = result.(Model)

	if !strings.Contains(m.browseStatus, "https://github.com/catppuccin/tmux") {
		t.Errorf("expected GitHub URL in status, got %q", m.browseStatus)
	}
}

func TestOpenFromBrowse_NonGitHubHost(t *testing.T) {
	m := newBrowseModel(t)
	m.browseRegistry.Plugins = append(m.browseRegistry.Plugins, registry.RegistryItem{
		Repo: "gitlab-user/tmux-theme", Description: "A theme", Category: "theme", Stars: 10, Host: "gitlab.com",
	})
	m.browseResults = m.browseRegistry.Plugins
	m.browseScroll.cursor = len(m.browseResults) - 1

	result, _ := m.openFromBrowse()
	m = result.(Model)

	if !strings.Contains(m.browseStatus, "https://gitlab.com/gitlab-user/tmux-theme") {
		t.Errorf("expected GitLab URL in status, got %q", m.browseStatus)
	}
}

func TestInstallFromBrowse_NonGitHubHost_UsesFullURL(t *testing.T) {
	m := newBrowseModel(t)
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "tmux.conf")
	os.WriteFile(confPath, []byte("# tmux config\n"), 0o644)
	m.cfg.TmuxConf = confPath

	m.browseRegistry = &registry.Registry{
		Categories: []string{"theme"},
		Plugins: []registry.RegistryItem{
			{Repo: "gitlab-user/tmux-theme", Description: "A theme", Category: "theme", Stars: 10, Host: "gitlab.com"},
		},
	}
	m.browseResults = m.browseRegistry.Plugins
	m.browseScroll.cursor = 0

	result, cmd := m.installFromBrowse()
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Fatalf("expected ScreenProgress, got %d", m.screen)
	}
	if m.totalItems != 1 {
		t.Errorf("expected 1 total item, got %d", m.totalItems)
	}
	if cmd == nil {
		t.Error("expected non-nil install command")
	}

	wantSpec := "https://gitlab.com/gitlab-user/tmux-theme"

	// Check that the plugin was added with the full URL spec.
	found := false
	for _, p := range m.plugins {
		if p.Spec == wantSpec {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected plugin with spec %q in plugins list", wantSpec)
	}

	data, _ := os.ReadFile(confPath)
	if !strings.Contains(string(data), wantSpec) {
		t.Errorf("expected full GitLab URL in tmux.conf, got:\n%s", data)
	}
}

func TestInstallFromBrowse_GitHubHost_UsesShorthand(t *testing.T) {
	m := newBrowseModel(t)
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "tmux.conf")
	os.WriteFile(confPath, []byte("# tmux config\n"), 0o644)
	m.cfg.TmuxConf = confPath

	m.browseScroll.cursor = 0

	result, cmd := m.installFromBrowse()
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Fatalf("expected ScreenProgress, got %d", m.screen)
	}
	if cmd == nil {
		t.Error("expected non-nil install command")
	}

	// Check plugin was added with shorthand spec (no URL prefix).
	found := false
	for _, p := range m.plugins {
		if p.Spec == "catppuccin/tmux" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected plugin with spec catppuccin/tmux in plugins list")
	}

	data, _ := os.ReadFile(confPath)
	content := string(data)
	if !strings.Contains(content, "catppuccin/tmux") {
		t.Errorf("expected shorthand repo in tmux.conf, got:\n%s", content)
	}
	if strings.Contains(content, "https://") {
		t.Errorf("expected no URL prefix for GitHub repo, got:\n%s", content)
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
