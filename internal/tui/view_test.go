package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

func TestViewList_ContainsPluginNames(t *testing.T) {
	plugins := []plugin.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
		{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank"},
	}
	m := newTestModel(t, plugins)
	m.width = 100
	m.viewHeight = 20

	view := m.View()
	for _, p := range plugins {
		if !strings.Contains(view, p.Name) {
			t.Errorf("expected view to contain plugin name %q", p.Name)
		}
	}
}

func TestViewList_EmptyPlugins(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 80
	m.viewHeight = 20

	view := m.View()
	if !strings.Contains(view, "No plugins configured") {
		t.Error("expected 'No plugins configured' message for empty plugin list")
	}
}

func TestViewList_WithOrphans(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 80
	m.viewHeight = 20
	m.orphans = []OrphanItem{
		{Name: "old-plugin", Path: "/tmp/old-plugin"},
		{Name: "stale-plugin", Path: "/tmp/stale-plugin"},
	}

	view := m.View()
	if !strings.Contains(view, "old-plugin") {
		t.Error("expected view to contain orphan name 'old-plugin'")
	}
	if !strings.Contains(view, "stale-plugin") {
		t.Error("expected view to contain orphan name 'stale-plugin'")
	}
	if !strings.Contains(view, "Orphaned") {
		t.Error("expected view to contain 'Orphaned' label")
	}
}

func TestViewList_HelpBar(t *testing.T) {
	// With not-installed plugin: help should show "install".
	m := newTestModel(t, nil)
	m.width = 100
	m.viewHeight = 20
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
	}

	view := m.View()
	if !strings.Contains(view, "quit") {
		t.Error("expected help bar to contain 'quit'")
	}
	if !strings.Contains(view, "install") {
		t.Error("expected help bar to contain 'install' when not-installed plugins exist")
	}

	// With only installed plugins: help should show "update" but not "install".
	m.plugins = []PluginItem{
		{Name: "b", Spec: "user/b", Status: StatusInstalled},
	}
	view = m.View()
	if !strings.Contains(view, "quit") {
		t.Error("expected help bar to contain 'quit'")
	}
	if !strings.Contains(view, "update") {
		t.Error("expected help bar to contain 'update' when installed plugins exist")
	}
}

func TestStatusSummary(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Status: StatusInstalled},
		{Name: "b", Status: StatusInstalled},
		{Name: "c", Status: StatusNotInstalled},
		{Name: "d", Status: StatusOutdated},
		{Name: "e", Status: StatusChecking},
	}

	summary := m.statusSummary()
	// 3 installed (StatusInstalled x2 + StatusChecking), 1 not installed, 1 outdated.
	if !strings.Contains(summary, "3 installed") {
		t.Errorf("expected '3 installed' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1 not installed") {
		t.Errorf("expected '1 not installed' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1 outdated") {
		t.Errorf("expected '1 outdated' in summary, got %q", summary)
	}
}

func TestViewProgress_ShowsOperation(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.totalItems = 2
	m.completedItems = 0
	m.processing = true
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Install") {
		t.Error("expected progress view to contain 'Install'")
	}
}

func TestViewProgress_ShowsCounter(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.totalItems = 5
	m.completedItems = 3
	m.processing = true
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Processing 3 of 5") {
		t.Errorf("expected 'Processing 3 of 5' in view, got:\n%s", view)
	}
}

func TestViewProgress_ShowsResultsWhenComplete(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.totalItems = 2
	m.completedItems = 2
	m.processing = false
	m.width = 80
	m.results = []ResultItem{
		{Name: "plugin-alpha", Success: true, Message: "installed"},
		{Name: "plugin-beta", Success: false, Message: "clone failed"},
	}

	view := m.View()
	if !strings.Contains(view, "plugin-alpha") {
		t.Error("expected results to contain 'plugin-alpha'")
	}
	if !strings.Contains(view, "plugin-beta") {
		t.Error("expected results to contain 'plugin-beta'")
	}
	if !strings.Contains(view, "clone failed") {
		t.Error("expected results to contain failure message 'clone failed'")
	}
}

func TestViewProgress_ShowsCurrentItem(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.totalItems = 2
	m.completedItems = 0
	m.processing = true
	m.currentItemName = "my-plugin"
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "my-plugin") {
		t.Error("expected progress view to contain current item name 'my-plugin'")
	}
}

func TestViewList_RenderStatus_AllStatuses(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 120
	m.viewHeight = 20
	m.plugins = []PluginItem{
		{Name: "p1", Status: StatusInstalled},
		{Name: "p2", Status: StatusNotInstalled},
		{Name: "p3", Status: StatusOutdated},
		{Name: "p4", Status: StatusCheckFailed},
	}

	view := m.View()
	if !strings.Contains(view, "Installed") {
		t.Error("expected view to contain 'Installed' status")
	}
	if !strings.Contains(view, "Not Installed") {
		t.Error("expected view to contain 'Not Installed' status")
	}
	if !strings.Contains(view, "Outdated") {
		t.Error("expected view to contain 'Outdated' status")
	}
}

func TestViewList_WithMultiSelect(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 100
	m.viewHeight = 20
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusInstalled},
		{Name: "beta", Status: StatusNotInstalled},
	}
	m.selected = map[int]bool{0: true}
	m.multiSelectActive = true

	view := m.View()
	if !strings.Contains(view, "alpha") {
		t.Error("expected view to contain 'alpha'")
	}
}

func TestCenterText_WithSizeKnown(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 80
	m.sizeKnown = true

	result := m.centerText("hello")
	// The result should be wider than the input due to centering.
	if lipgloss.Width(result) <= lipgloss.Width("hello") {
		t.Error("expected centered text to be padded wider than input")
	}
}

func TestCenterText_WithoutSizeKnown(t *testing.T) {
	m := newTestModel(t, nil)
	m.sizeKnown = false

	result := m.centerText("hello")
	if result != "hello" {
		t.Errorf("expected unchanged text when size not known, got %q", result)
	}
}

func TestCenterBlock_WithSizeKnown(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 80
	m.sizeKnown = true

	block := "line1\nline2"
	result := m.centerBlock(block)
	if lipgloss.Width(result) <= lipgloss.Width(block) {
		t.Error("expected centered block to be padded wider than input")
	}
}

func TestCenterBlock_WithoutSizeKnown(t *testing.T) {
	m := newTestModel(t, nil)
	m.sizeKnown = false

	block := "line1\nline2"
	result := m.centerBlock(block)
	if result != block {
		t.Errorf("expected unchanged block when size not known, got %q", result)
	}
}

func TestCenterText_WideTextUnchanged(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 20
	m.sizeKnown = true

	// Text wider than content area (20 - 4 = 16) should be returned unchanged.
	wide := strings.Repeat("x", 20)
	result := m.centerText(wide)
	if result != wide {
		t.Error("expected wide text to be returned unchanged")
	}
}

func TestCenterBlock_WideBlockUnchanged(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 20
	m.sizeKnown = true

	wide := strings.Repeat("x", 20)
	result := m.centerBlock(wide)
	if result != wide {
		t.Error("expected wide block to be returned unchanged")
	}
}

func TestViewList_HelpBar_WithOrphans(t *testing.T) {
	m := newTestModel(t, nil)
	m.width = 100
	m.viewHeight = 20
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusInstalled},
	}
	m.orphans = []OrphanItem{
		{Name: "orphan", Path: "/tmp/orphan"},
	}

	view := m.View()
	if !strings.Contains(view, "clean") {
		t.Error("expected help bar to contain 'clean' when orphans exist")
	}
}

func TestViewProgress_AutoOp_ShowsQuitOnly(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.autoOp = OpInstall
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 80
	m.results = []ResultItem{
		{Name: "test-plugin", Success: true, Message: "installed"},
	}

	view := m.View()
	if !strings.Contains(view, "quit") {
		t.Error("expected progress view to contain 'quit'")
	}
	if strings.Contains(view, "back to list") {
		t.Error("expected progress view NOT to contain 'back to list' in auto-op mode")
	}
}

func TestViewProgress_NoAutoOp_ShowsBackToList(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 80
	m.results = []ResultItem{
		{Name: "test-plugin", Success: true, Message: "installed"},
	}

	view := m.View()
	if !strings.Contains(view, "back to list") {
		t.Error("expected progress view to contain 'back to list' when no auto-op")
	}
}

func TestViewProgress_ShowsCommitCount(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 100
	m.results = []ResultItem{
		{
			Name:    "tmux-sensible",
			Success: true,
			Message: "updated",
			Commits: []git.Commit{
				{Hash: "abc1234", Message: "add feature"},
				{Hash: "def5678", Message: "fix bug"},
			},
		},
	}

	view := m.View()
	if !strings.Contains(view, "2 new commits") {
		t.Error("expected view to show '2 new commits'")
	}
	if !strings.Contains(view, "▸") {
		t.Error("expected collapsed indicator '▸' for plugin with commits")
	}
}

func TestViewProgress_ShowsSingleCommitCount(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 100
	m.results = []ResultItem{
		{
			Name:    "tmux-yank",
			Success: true,
			Message: "updated",
			Commits: []git.Commit{
				{Hash: "abc1234", Message: "add feature"},
			},
		},
	}

	view := m.View()
	if !strings.Contains(view, "1 new commit)") {
		t.Error("expected view to show '1 new commit)' (singular)")
	}
}

func TestViewProgress_NoCommitsNoIndicator(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 100
	m.results = []ResultItem{
		{Name: "tmux-yank", Success: true, Message: "updated"},
	}

	view := m.View()
	if strings.Contains(view, "▸") || strings.Contains(view, "▼") {
		t.Error("expected no expand/collapse indicator for plugin without commits")
	}
	if strings.Contains(view, "new commit") {
		t.Error("expected no commit count for plugin without commits")
	}
}

func TestViewProgress_HelpShowsViewCommits(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.totalItems = 1
	m.completedItems = 1
	m.processing = false
	m.width = 100
	m.results = []ResultItem{
		{Name: "test", Success: true},
	}

	view := m.View()
	if !strings.Contains(view, "view commits") {
		t.Error("expected help to contain 'view commits'")
	}
}
