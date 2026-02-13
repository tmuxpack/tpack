package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleCheckResult_Outdated(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusChecking},
	}

	msg := pluginCheckResultMsg{Name: "alpha", Outdated: true}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.plugins[0].Status != StatusOutdated {
		t.Errorf("expected StatusOutdated, got %s", m.plugins[0].Status)
	}
}

func TestHandleCheckResult_Error(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusChecking},
	}

	msg := pluginCheckResultMsg{Name: "alpha", Err: errors.New("fetch failed")}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.plugins[0].Status != StatusCheckFailed {
		t.Errorf("expected StatusCheckFailed, got %s", m.plugins[0].Status)
	}
}

func TestHandleCheckResult_UpToDate(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusChecking},
	}

	msg := pluginCheckResultMsg{Name: "alpha", Outdated: false, Err: nil}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.plugins[0].Status != StatusInstalled {
		t.Errorf("expected StatusInstalled, got %s", m.plugins[0].Status)
	}
}

func TestHandleCheckResult_UnknownPlugin(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusChecking},
	}

	// Should not panic when the plugin name doesn't match any known plugin.
	msg := pluginCheckResultMsg{Name: "nonexistent", Outdated: true}
	result, _ := m.Update(msg)
	_ = result.(Model) // should not panic
}

func TestHandleInstallResult_Success(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusNotInstalled},
	}
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.processing = true
	m.totalItems = 1
	m.completedItems = 0

	msg := pluginInstallResultMsg{Name: "alpha", Success: true, Message: "installed"}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.plugins[0].Status != StatusInstalled {
		t.Errorf("expected StatusInstalled after successful install, got %s", m.plugins[0].Status)
	}
	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.results))
	}
	if !m.results[0].Success {
		t.Error("expected result to be successful")
	}
}

func TestHandleInstallResult_Failure(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusNotInstalled},
	}
	m.screen = ScreenProgress
	m.operation = OpInstall
	m.processing = true
	m.totalItems = 1
	m.completedItems = 0

	msg := pluginInstallResultMsg{Name: "alpha", Success: false, Message: "clone failed"}
	result, _ := m.Update(msg)
	m = result.(Model)

	// Status should remain unchanged on failure.
	if m.plugins[0].Status != StatusNotInstalled {
		t.Errorf("expected StatusNotInstalled after failed install, got %s", m.plugins[0].Status)
	}
	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.results))
	}
	if m.results[0].Success {
		t.Error("expected result to be failure")
	}
}

func TestHandleUpdateResult(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusInstalled},
	}
	m.screen = ScreenProgress
	m.operation = OpUpdate
	m.processing = true
	m.totalItems = 1
	m.completedItems = 0

	msg := pluginUpdateResultMsg{Name: "alpha", Success: true, Message: "updated"}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.completedItems != 1 {
		t.Errorf("expected completedItems=1, got %d", m.completedItems)
	}
	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.results))
	}
}

func TestHandleCleanResult(t *testing.T) {
	m := newTestModel(t, nil)
	m.orphans = []OrphanItem{
		{Name: "orphan-a", Path: "/tmp/orphan-a"},
	}
	m.screen = ScreenProgress
	m.operation = OpClean
	m.processing = true
	m.totalItems = 1
	m.completedItems = 0

	msg := pluginCleanResultMsg{Name: "orphan-a", Success: true, Message: "removed"}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.completedItems != 1 {
		t.Errorf("expected completedItems=1, got %d", m.completedItems)
	}
	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.results))
	}
}

func TestHandleUninstallResult_Success(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusInstalled},
	}
	m.screen = ScreenProgress
	m.operation = OpUninstall
	m.processing = true
	m.totalItems = 1
	m.completedItems = 0

	msg := pluginUninstallResultMsg{Name: "alpha", Success: true, Message: "removed"}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.plugins[0].Status != StatusNotInstalled {
		t.Errorf("expected StatusNotInstalled after uninstall, got %s", m.plugins[0].Status)
	}
}

func TestReturnToList_RemovesCleanedOrphans(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.operation = OpClean
	m.processing = false
	m.orphans = []OrphanItem{
		{Name: "cleaned", Path: "/tmp/cleaned"},
		{Name: "remaining", Path: "/tmp/remaining"},
	}
	m.results = []ResultItem{
		{Name: "cleaned", Success: true, Message: "removed"},
	}

	m = m.returnToList()

	if len(m.orphans) != 1 {
		t.Fatalf("expected 1 remaining orphan, got %d", len(m.orphans))
	}
	if m.orphans[0].Name != "remaining" {
		t.Errorf("expected remaining orphan 'remaining', got %q", m.orphans[0].Name)
	}
}

func TestReturnToList_ClampsNegativeCursor(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = nil
	m.cursor = 5
	m.screen = ScreenProgress
	m.operation = OpClean
	m.processing = false

	m = m.returnToList()

	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestMoveCursorDown_Scrolling(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = make([]PluginItem, 20)
	for i := range m.plugins {
		m.plugins[i] = PluginItem{Name: "plugin", Status: StatusInstalled}
	}
	m.viewHeight = 5
	m.scrollOffset = 0
	m.cursor = 0

	// Move cursor down past the scroll threshold (viewHeight - ScrollOffsetMargin = 5-3 = 2).
	for i := 0; i < 3; i++ {
		m.moveCursorDown()
	}

	if m.scrollOffset == 0 {
		t.Error("expected scrollOffset to increase when cursor moves past visible area")
	}
	if m.cursor != 3 {
		t.Errorf("expected cursor at 3, got %d", m.cursor)
	}
}

func TestMoveCursorUp_Scrolling(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = make([]PluginItem, 20)
	for i := range m.plugins {
		m.plugins[i] = PluginItem{Name: "plugin", Status: StatusInstalled}
	}
	m.viewHeight = 5
	m.scrollOffset = 10
	m.cursor = 12

	// Move cursor up into the scroll margin zone.
	m.moveCursorUp()
	m.moveCursorUp()

	if m.scrollOffset >= 10 {
		t.Error("expected scrollOffset to decrease when cursor moves above visible area")
	}
}

func TestHasCheckingPlugins(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Status: StatusInstalled},
		{Name: "b", Status: StatusChecking},
	}

	if !m.hasCheckingPlugins() {
		t.Error("expected hasCheckingPlugins() to return true when a plugin is checking")
	}

	m.plugins = []PluginItem{
		{Name: "a", Status: StatusInstalled},
		{Name: "b", Status: StatusOutdated},
	}

	if m.hasCheckingPlugins() {
		t.Error("expected hasCheckingPlugins() to return false when no plugin is checking")
	}
}

func TestUpdateProgress_IgnoresKeysWhileProcessing(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.processing = true
	m.operation = OpInstall
	m.totalItems = 1
	m.completedItems = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	result, cmd := m.Update(msg)
	updated := result.(Model)

	// While processing, key presses (except force quit) should be ignored.
	if cmd != nil {
		t.Error("expected nil command while processing, got non-nil")
	}
	if updated.screen != ScreenProgress {
		t.Errorf("expected to stay on ScreenProgress, got %d", updated.screen)
	}
}

func TestUpdateProgress_EscReturnsToList(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.processing = false
	m.operation = OpInstall

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.Update(msg)
	updated := result.(Model)

	if updated.screen != ScreenList {
		t.Errorf("expected ScreenList after esc, got %d", updated.screen)
	}
}

func TestUpdateProgress_QuitWhenNotProcessing(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.processing = false
	m.operation = OpInstall

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command when q pressed on progress screen while not processing")
	}
}

func TestUpdateList_ToggleSelection(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Status: StatusInstalled},
		{Name: "beta", Status: StatusNotInstalled},
	}
	m.viewHeight = 10

	// Press tab to toggle selection.
	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := m.Update(msg)
	m = result.(Model)

	if !m.selected[0] {
		t.Error("expected plugin at cursor to be selected after tab")
	}
	if !m.multiSelectActive {
		t.Error("expected multiSelectActive to be true after tab")
	}
}

func TestUpdateList_InstallKey(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusNotInstalled},
	}
	m.viewHeight = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress after 'i', got %d", m.screen)
	}
	if m.operation != OpInstall {
		t.Errorf("expected OpInstall, got %d", m.operation)
	}
	if cmd == nil {
		t.Error("expected non-nil command after install")
	}
}

func TestUpdateList_UpdateKey(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusInstalled},
	}
	m.viewHeight = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress after 'u', got %d", m.screen)
	}
	if m.operation != OpUpdate {
		t.Errorf("expected OpUpdate, got %d", m.operation)
	}
	if cmd == nil {
		t.Error("expected non-nil command after update")
	}
}

func TestUpdateList_CleanKey(t *testing.T) {
	m := newTestModel(t, nil)
	m.orphans = []OrphanItem{
		{Name: "orphan", Path: "/tmp/orphan"},
	}
	m.viewHeight = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress after 'c', got %d", m.screen)
	}
	if m.operation != OpClean {
		t.Errorf("expected OpClean, got %d", m.operation)
	}
	if cmd == nil {
		t.Error("expected non-nil command after clean")
	}
}

func TestUpdateList_UninstallKey(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "alpha", Spec: "user/alpha", Status: StatusInstalled},
	}
	m.viewHeight = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	result, cmd := m.Update(msg)
	m = result.(Model)

	if m.screen != ScreenProgress {
		t.Errorf("expected ScreenProgress after 'x', got %d", m.screen)
	}
	if m.operation != OpUninstall {
		t.Errorf("expected OpUninstall, got %d", m.operation)
	}
	if cmd == nil {
		t.Error("expected non-nil command after uninstall")
	}
}

func TestBuildCleanOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.orphans = []OrphanItem{
		{Name: "orphan-a", Path: "/tmp/orphan-a"},
		{Name: "orphan-b", Path: "/tmp/orphan-b"},
	}

	ops := m.buildCleanOps()
	if len(ops) != 2 {
		t.Errorf("expected 2 clean ops, got %d", len(ops))
	}
	if ops[0].Name != "orphan-a" {
		t.Errorf("expected first op name 'orphan-a', got %q", ops[0].Name)
	}
	if ops[1].Name != "orphan-b" {
		t.Errorf("expected second op name 'orphan-b', got %q", ops[1].Name)
	}
}

func TestStartOperation_OpNone(t *testing.T) {
	m := newTestModel(t, nil)
	result, cmd := m.startOperation(OpNone)
	updated := result.(Model)

	if updated.screen != ScreenList {
		t.Errorf("expected to stay on ScreenList for OpNone, got %d", updated.screen)
	}
	if cmd != nil {
		t.Error("expected nil command for OpNone")
	}
}

func TestForceQuit_FromList(t *testing.T) {
	m := newTestModel(t, nil)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command on ctrl+c")
	}
}

func TestForceQuit_FromProgress(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenProgress
	m.processing = true

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command on ctrl+c even while processing")
	}
}

func TestWindowSizeMsg_MinViewHeight(t *testing.T) {
	m := newTestModel(t, nil)
	// Set height so small that viewHeight would be less than MinViewHeight.
	msg := tea.WindowSizeMsg{Width: 80, Height: 5}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.viewHeight < MinViewHeight {
		t.Errorf("expected viewHeight >= %d, got %d", MinViewHeight, m.viewHeight)
	}
}

func TestWindowSizeMsg_ProgressBarWidthCapped(t *testing.T) {
	m := newTestModel(t, nil)
	msg := tea.WindowSizeMsg{Width: 200, Height: 40}
	result, _ := m.Update(msg)
	m = result.(Model)

	if m.progressBar.Width > ProgressBarMaxWidth {
		t.Errorf("expected progressBar.Width <= %d, got %d", ProgressBarMaxWidth, m.progressBar.Width)
	}
}

func TestMoveCursorDown_AtEnd(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Status: StatusInstalled},
		{Name: "b", Status: StatusInstalled},
	}
	m.viewHeight = 10
	m.cursor = 1 // already at last plugin

	m.moveCursorDown()
	if m.cursor != 1 {
		t.Errorf("expected cursor to remain at 1 when at end, got %d", m.cursor)
	}
}

func TestMoveCursorUp_AtBeginning(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Status: StatusInstalled},
	}
	m.viewHeight = 10
	m.cursor = 0

	m.moveCursorUp()
	if m.cursor != 0 {
		t.Errorf("expected cursor to remain at 0 when at beginning, got %d", m.cursor)
	}
}
