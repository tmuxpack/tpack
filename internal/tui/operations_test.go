package tui

import (
	"errors"
	"testing"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestInstallPluginCmd_Success(t *testing.T) {
	cloner := git.NewMockCloner()
	op := pendingOp{
		Name: "test-plugin",
		Spec: "user/test-plugin",
		Path: t.TempDir() + "/test-plugin/",
	}

	cmd := installPluginCmd(cloner, op)
	msg := cmd()

	result, ok := msg.(pluginInstallResultMsg)
	if !ok {
		t.Fatalf("expected pluginInstallResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", result.Name)
	}
}

func TestInstallPluginCmd_Failure(t *testing.T) {
	cloner := git.NewMockCloner()
	cloner.Err = errors.New("clone failed")
	op := pendingOp{
		Name: "test-plugin",
		Spec: "user/test-plugin",
		Path: t.TempDir() + "/test-plugin/",
	}

	cmd := installPluginCmd(cloner, op)
	msg := cmd()

	result, ok := msg.(pluginInstallResultMsg)
	if !ok {
		t.Fatalf("expected pluginInstallResultMsg, got %T", msg)
	}
	if result.Success {
		t.Error("expected failure, got success")
	}
}

func TestUpdatePluginCmd_Success(t *testing.T) {
	puller := git.NewMockPuller()
	puller.Output = "Already up to date."
	dir := t.TempDir()
	op := pendingOp{
		Name: "test-plugin",
		Path: dir + "/",
	}

	cmd := updatePluginCmd(puller, op)
	msg := cmd()

	result, ok := msg.(pluginUpdateResultMsg)
	if !ok {
		t.Fatalf("expected pluginUpdateResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Output != "Already up to date." {
		t.Errorf("expected output 'Already up to date.', got %q", result.Output)
	}
}

func TestUpdatePluginCmd_Failure(t *testing.T) {
	puller := git.NewMockPuller()
	puller.Err = errors.New("pull failed")
	op := pendingOp{
		Name: "test-plugin",
		Path: t.TempDir() + "/",
	}

	cmd := updatePluginCmd(puller, op)
	msg := cmd()

	result, ok := msg.(pluginUpdateResultMsg)
	if !ok {
		t.Fatalf("expected pluginUpdateResultMsg, got %T", msg)
	}
	if result.Success {
		t.Error("expected failure, got success")
	}
}

func TestCleanPluginCmd_Success(t *testing.T) {
	dir := t.TempDir()
	op := pendingOp{
		Name: "orphan-plugin",
		Path: dir,
	}

	cmd := cleanPluginCmd(op)
	msg := cmd()

	result, ok := msg.(pluginCleanResultMsg)
	if !ok {
		t.Fatalf("expected pluginCleanResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
}

func TestCleanPluginCmd_NonExistentDir(t *testing.T) {
	op := pendingOp{
		Name: "ghost-plugin",
		Path: "/tmp/nonexistent-tpm-test-dir-12345/",
	}

	cmd := cleanPluginCmd(op)
	msg := cmd()

	result, ok := msg.(pluginCleanResultMsg)
	if !ok {
		t.Fatalf("expected pluginCleanResultMsg, got %T", msg)
	}
	// RemoveAll on nonexistent path succeeds.
	if !result.Success {
		t.Errorf("expected success for nonexistent dir, got failure: %s", result.Message)
	}
}

func TestUninstallPluginCmd_Success(t *testing.T) {
	dir := t.TempDir()
	op := pendingOp{
		Name: "test-plugin",
		Path: dir,
	}

	cmd := uninstallPluginCmd(op)
	msg := cmd()

	result, ok := msg.(pluginUninstallResultMsg)
	if !ok {
		t.Fatalf("expected pluginUninstallResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", result.Name)
	}
}

func TestBuildUninstallOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusInstalled},
		{Name: "b", Spec: "user/b", Status: StatusNotInstalled},
		{Name: "c", Spec: "user/c", Status: StatusInstalled},
	}
	m.cursor = 0

	ops := m.buildUninstallOps()
	if len(ops) != 1 {
		t.Errorf("expected 1 uninstall op (cursor on installed), got %d", len(ops))
	}
	if ops[0].Name != "a" {
		t.Errorf("expected op name 'a', got %s", ops[0].Name)
	}
}

func TestBuildUninstallOps_SkipsNotInstalled(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
	}
	m.cursor = 0

	ops := m.buildUninstallOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 uninstall ops for not-installed plugin, got %d", len(ops))
	}
}

func TestBuildInstallOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
		{Name: "b", Spec: "user/b", Status: StatusInstalled},
		{Name: "c", Spec: "user/c", Status: StatusNotInstalled},
	}
	m.cursor = 0

	ops := m.buildInstallOps()
	if len(ops) != 1 {
		t.Errorf("expected 1 install op (cursor on not-installed), got %d", len(ops))
	}
}

func TestBuildInstallOps_MultiSelect(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
		{Name: "b", Spec: "user/b", Status: StatusInstalled},
		{Name: "c", Spec: "user/c", Status: StatusNotInstalled},
	}
	m.selected = map[int]bool{0: true, 2: true}
	m.multiSelectActive = true

	ops := m.buildInstallOps()
	if len(ops) != 2 {
		t.Errorf("expected 2 install ops, got %d", len(ops))
	}
}

func TestBuildUpdateOps_AllInstalled(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusInstalled},
		{Name: "b", Spec: "user/b", Status: StatusInstalled},
	}
	m.cursor = 0

	ops := m.buildUpdateOps()
	// Cursor on installed plugin → 1 op. But if nothing selected and cursor match
	// is installed, it returns just cursor. Then fallback to all installed.
	if len(ops) != 1 {
		t.Errorf("expected 1 update op (cursor), got %d", len(ops))
	}
}

func TestDispatchNext_EmptyQueue(t *testing.T) {
	m := newTestModel(t, nil)
	m.pendingItems = nil

	cmd := m.dispatchNext()
	if cmd != nil {
		t.Error("expected nil cmd for empty queue")
	}
	if m.processing {
		t.Error("expected processing to be false")
	}
}

func TestBuildAutoInstallOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
		{Name: "b", Spec: "user/b", Status: StatusInstalled},
		{Name: "c", Spec: "user/c", Status: StatusNotInstalled},
	}

	ops := m.buildAutoInstallOps()
	if len(ops) != 2 {
		t.Errorf("expected 2 auto install ops, got %d", len(ops))
	}
	if ops[0].Name != "a" {
		t.Errorf("expected first op name 'a', got %s", ops[0].Name)
	}
	if ops[1].Name != "c" {
		t.Errorf("expected second op name 'c', got %s", ops[1].Name)
	}
}

func TestBuildAutoInstallOps_NoneNotInstalled(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusInstalled},
	}

	ops := m.buildAutoInstallOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 auto install ops, got %d", len(ops))
	}
}

func TestBuildAutoUpdateOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusInstalled},
		{Name: "b", Spec: "user/b", Status: StatusNotInstalled},
		{Name: "c", Spec: "user/c", Status: StatusOutdated},
		{Name: "d", Spec: "user/d", Status: StatusChecking},
	}

	ops := m.buildAutoUpdateOps()
	if len(ops) != 3 {
		t.Errorf("expected 3 auto update ops (installed+outdated+checking), got %d", len(ops))
	}
}

func TestBuildAutoUpdateOps_NoneInstalled(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		{Name: "a", Spec: "user/a", Status: StatusNotInstalled},
	}

	ops := m.buildAutoUpdateOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 auto update ops, got %d", len(ops))
	}
}

func TestDispatchNext_WithRunner_SourcesOnInstall(t *testing.T) {
	runner := tmux.NewMockRunner()
	cfg := &config.Config{PluginPath: t.TempDir() + "/", TmuxConf: "/tmp/test.conf"}
	deps := Deps{
		Cloner:    git.NewMockCloner(),
		Puller:    git.NewMockPuller(),
		Validator: git.NewMockValidator(),
		Fetcher:   git.NewMockFetcher(),
		Runner:    runner,
	}
	m := NewModel(cfg, nil, deps)
	m.operation = OpInstall
	m.pendingItems = nil

	cmd := m.dispatchNext()
	if cmd == nil {
		t.Fatal("expected source command when runner is set and install completes")
	}

	// Execute the command and verify it calls SourceFile.
	msg := cmd()
	if _, ok := msg.(sourceCompleteMsg); !ok {
		t.Fatalf("expected sourceCompleteMsg, got %T", msg)
	}

	found := false
	for _, c := range runner.Calls {
		if c.Method == "SourceFile" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected SourceFile to be called")
	}
}

func TestDispatchNext_WithRunner_SourcesOnUpdate(t *testing.T) {
	runner := tmux.NewMockRunner()
	cfg := &config.Config{PluginPath: t.TempDir() + "/", TmuxConf: "/tmp/test.conf"}
	deps := Deps{
		Cloner:    git.NewMockCloner(),
		Puller:    git.NewMockPuller(),
		Validator: git.NewMockValidator(),
		Fetcher:   git.NewMockFetcher(),
		Runner:    runner,
	}
	m := NewModel(cfg, nil, deps)
	m.operation = OpUpdate
	m.pendingItems = nil

	cmd := m.dispatchNext()
	if cmd == nil {
		t.Fatal("expected source command when runner is set and update completes")
	}
}

func TestDispatchNext_WithRunner_NoSourceOnClean(t *testing.T) {
	runner := tmux.NewMockRunner()
	cfg := &config.Config{PluginPath: t.TempDir() + "/", TmuxConf: "/tmp/test.conf"}
	deps := Deps{
		Cloner:    git.NewMockCloner(),
		Puller:    git.NewMockPuller(),
		Validator: git.NewMockValidator(),
		Fetcher:   git.NewMockFetcher(),
		Runner:    runner,
	}
	m := NewModel(cfg, nil, deps)
	m.operation = OpClean
	m.pendingItems = nil

	cmd := m.dispatchNext()
	if cmd != nil {
		t.Error("expected nil command for clean operation (no sourcing needed)")
	}
}

func TestDispatchNext_NoRunner_NoSource(t *testing.T) {
	m := newTestModel(t, nil)
	m.operation = OpInstall
	m.pendingItems = nil

	cmd := m.dispatchNext()
	if cmd != nil {
		t.Error("expected nil command when no runner is set")
	}
}

func TestSourceCmd(t *testing.T) {
	runner := tmux.NewMockRunner()
	cmd := sourceCmd(runner, "/tmp/test.conf")
	msg := cmd()

	result, ok := msg.(sourceCompleteMsg)
	if !ok {
		t.Fatalf("expected sourceCompleteMsg, got %T", msg)
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}

	if len(runner.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.Calls))
	}
	if runner.Calls[0].Method != "SourceFile" {
		t.Errorf("expected SourceFile call, got %s", runner.Calls[0].Method)
	}
	if runner.Calls[0].Args[0] != "/tmp/test.conf" {
		t.Errorf("expected conf path /tmp/test.conf, got %s", runner.Calls[0].Args[0])
	}
}
