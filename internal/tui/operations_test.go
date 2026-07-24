package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestBuildInstallOpsUsesDirectoryKeyAndPreservesRaw(t *testing.T) {
	rootDir := t.TempDir()
	m := newTestModel(t, nil)
	m.cfg.PluginPath = mustRoot(t, rootDir)
	m.plugins = []PluginItem{
		{Raw: "catppuccin/tmux#v2", Name: "tmux", DirName: "tmux-87a1216f1f68", Spec: "catppuccin/tmux", Branch: "v2", Status: StatusNotInstalled},
		{Raw: "dracula/tmux alias=dark", Name: "tmux", DirName: "dark", Spec: "dracula/tmux", Status: StatusNotInstalled},
	}
	m.selected = map[int]bool{0: true, 1: true}
	m.multiSelectActive = true

	ops := m.buildInstallOps()
	if len(ops) != 2 {
		t.Fatalf("install operations = %d, want 2", len(ops))
	}
	if ops[0].Path != filepath.Join(rootDir, "tmux-87a1216f1f68") || ops[1].Path != filepath.Join(rootDir, "dark") {
		t.Fatalf("operation paths = %q, %q", ops[0].Path, ops[1].Path)
	}
	if ops[0].Raw != "catppuccin/tmux#v2" || ops[1].Raw != "dracula/tmux alias=dark" {
		t.Fatalf("operation raw specs = %q, %q", ops[0].Raw, ops[1].Raw)
	}
	if ops[0].DirName != "tmux-87a1216f1f68" || ops[1].DirName != "dark" {
		t.Fatalf("operation directory keys = %q, %q", ops[0].DirName, ops[1].DirName)
	}
}

func TestUninstallPluginCmdResolvesDirectoryKey(t *testing.T) {
	rootDir := t.TempDir()
	wantRemoved := filepath.Join(rootDir, "tmux-e74ab6318c07")
	wantKept := filepath.Join(rootDir, "tmux-87a1216f1f68")
	for _, dir := range []string{wantRemoved, wantKept} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	op := pendingOp{Name: "tmux", DirName: "tmux-e74ab6318c07", Path: wantRemoved, Root: mustRoot(t, rootDir)}

	msg := uninstallPluginCmd(op)()
	result, ok := msg.(pluginUninstallResultMsg)
	if !ok || !result.Success {
		t.Fatalf("uninstall result = %#v", msg)
	}
	if _, err := os.Stat(wantRemoved); !os.IsNotExist(err) {
		t.Fatalf("directory-keyed path was not removed: %v", err)
	}
	if _, err := os.Stat(wantKept); err != nil {
		t.Fatalf("other same-basename repository was changed: %v", err)
	}
}

func TestStartRemoveUsesRawConfigDeclaration(t *testing.T) {
	rootDir := t.TempDir()
	confPath := filepath.Join(t.TempDir(), "tmux.conf")
	const raw = "catppuccin/tmux alias=catppuccin#v2"
	if err := os.WriteFile(confPath, []byte("set -g @plugin \""+raw+"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newTestModel(t, nil)
	m.cfg.PluginPath = mustRoot(t, rootDir)
	m.cfg.TmuxConf = confPath
	m.plugins = []PluginItem{{Raw: raw, Name: "catppuccin", DirName: "catppuccin", Spec: "catppuccin/tmux", Branch: "v2", Status: StatusInstalled}}

	_, cmd := m.startOperation(OpRemove)
	if cmd == nil {
		t.Fatal("remove did not queue filesystem operation")
	}
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), raw) {
		t.Fatalf("raw config declaration was not removed:\n%s", data)
	}
}

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
	revParser := git.NewMockRevParser()
	revParser.Hash = "abc123"
	logger := git.NewMockLogger()
	dir := t.TempDir()
	op := pendingOp{
		Name: "test-plugin",
		Path: dir + "/",
	}

	cmd := updatePluginCmd(puller, revParser, logger, op)
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
	// Same hash before/after → no commits.
	if len(result.Commits) != 0 {
		t.Errorf("expected 0 commits when hash unchanged, got %d", len(result.Commits))
	}
}

func TestUpdatePluginCmd_WithCommits(t *testing.T) {
	puller := git.NewMockPuller()
	puller.Output = "Updating abc..def"

	callCount := 0
	revParser := &sequentialMockRevParser{
		hashes: []string{"abc123", "def456"},
		count:  &callCount,
	}
	logger := git.NewMockLogger()
	logger.Commits = []git.Commit{
		{Hash: "def456", Message: "add feature"},
		{Hash: "ccc333", Message: "fix bug"},
	}

	op := pendingOp{
		Name: "test-plugin",
		Path: t.TempDir() + "/",
	}

	cmd := updatePluginCmd(puller, revParser, logger, op)
	msg := cmd()

	result, ok := msg.(pluginUpdateResultMsg)
	if !ok {
		t.Fatalf("expected pluginUpdateResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if len(result.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(result.Commits))
	}
	if result.Commits[0].Hash != "def456" {
		t.Errorf("expected first commit hash def456, got %s", result.Commits[0].Hash)
	}
	if result.Dir != op.Path {
		t.Errorf("expected Dir=%q, got %q", op.Path, result.Dir)
	}
	if result.BeforeRef != "abc123" {
		t.Errorf("expected BeforeRef=abc123, got %q", result.BeforeRef)
	}
	if result.AfterRef != "def456" {
		t.Errorf("expected AfterRef=def456, got %q", result.AfterRef)
	}
}

func TestUpdatePluginCmd_NilRevParser(t *testing.T) {
	puller := git.NewMockPuller()
	puller.Output = "Already up to date."

	op := pendingOp{
		Name: "test-plugin",
		Path: t.TempDir() + "/",
	}

	cmd := updatePluginCmd(puller, nil, nil, op)
	msg := cmd()

	result, ok := msg.(pluginUpdateResultMsg)
	if !ok {
		t.Fatalf("expected pluginUpdateResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if len(result.Commits) != 0 {
		t.Errorf("expected 0 commits with nil revParser, got %d", len(result.Commits))
	}
}

func TestUpdatePluginCmd_Failure(t *testing.T) {
	puller := git.NewMockPuller()
	puller.Err = errors.New("pull failed")
	op := pendingOp{
		Name: "test-plugin",
		Path: t.TempDir() + "/",
	}

	cmd := updatePluginCmd(puller, nil, nil, op)
	msg := cmd()

	result, ok := msg.(pluginUpdateResultMsg)
	if !ok {
		t.Fatalf("expected pluginUpdateResultMsg, got %T", msg)
	}
	if result.Success {
		t.Error("expected failure, got success")
	}
}

// sequentialMockRevParser returns different hashes on sequential calls.
type sequentialMockRevParser struct {
	hashes []string
	count  *int
}

func (s *sequentialMockRevParser) RevParse(_ context.Context, _ string) (string, error) {
	idx := *s.count
	*s.count++
	if idx < len(s.hashes) {
		return s.hashes[idx], nil
	}
	return "unknown", nil
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

func TestBuildCleanOpsUsesOrphanDirectoryKey(t *testing.T) {
	m := newTestModel(t, nil)
	m.orphans = []OrphanItem{{
		Name: "display label", DirName: "orphan-directory", Path: "/plugins/orphan-directory",
	}}

	ops := m.buildCleanOps()

	if len(ops) != 1 || ops[0].Name != "display label" || ops[0].DirName != "orphan-directory" {
		t.Fatalf("clean operations = %+v", ops)
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
	rootDir := t.TempDir()
	dir := filepath.Join(rootDir, "test-plugin")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	op := pendingOp{
		Name:    "test-plugin",
		DirName: "test-plugin",
		Path:    dir,
		Root:    mustRoot(t, rootDir),
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

func TestRemovePluginDirCmd_Success(t *testing.T) {
	rootDir := t.TempDir()
	dir := filepath.Join(rootDir, "test-plugin")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	op := pendingOp{
		Name:    "test-plugin",
		DirName: "test-plugin",
		Path:    dir,
		Root:    mustRoot(t, rootDir),
	}

	cmd := removePluginDirCmd(op)
	msg := cmd()

	result, ok := msg.(pluginRemoveResultMsg)
	if !ok {
		t.Fatalf("expected pluginRemoveResultMsg, got %T", msg)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", result.Name)
	}
}

func TestRemovePluginDirCmd_NonExistentDir(t *testing.T) {
	rootDir := t.TempDir()
	op := pendingOp{
		Name:    "ghost-plugin",
		DirName: "ghost-plugin",
		Path:    filepath.Join(rootDir, "ghost-plugin"),
		Root:    mustRoot(t, rootDir),
	}

	cmd := removePluginDirCmd(op)
	msg := cmd()

	result, ok := msg.(pluginRemoveResultMsg)
	if !ok {
		t.Fatalf("expected pluginRemoveResultMsg, got %T", msg)
	}
	// RemoveAll on nonexistent path succeeds.
	if !result.Success {
		t.Errorf("expected success for nonexistent dir, got failure: %s", result.Message)
	}
}

func TestBuildRemoveOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusInstalled),
		testPluginItem("b", "user/b", StatusNotInstalled),
		testPluginItem("c", "user/c", StatusInstalled),
	}
	m.listScroll.cursor = 1

	ops := m.buildRemoveOps()
	if len(ops) != 1 {
		t.Fatalf("expected 1 remove op (cursor), got %d", len(ops))
	}
	if ops[0].Name != "b" {
		t.Errorf("expected op name 'b', got %s", ops[0].Name)
	}
}

func TestBuildRemoveOps_IncludesAllStatuses(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusInstalled),
		testPluginItem("b", "user/b", StatusNotInstalled),
	}
	m.selected = map[int]bool{0: true, 1: true}
	m.multiSelectActive = true

	ops := m.buildRemoveOps()
	if len(ops) != 2 {
		t.Errorf("expected 2 remove ops (both statuses), got %d", len(ops))
	}
}

func TestBuildUninstallOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusInstalled),
		testPluginItem("b", "user/b", StatusNotInstalled),
		testPluginItem("c", "user/c", StatusInstalled),
	}
	m.listScroll.cursor = 0

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
		testPluginItem("a", "user/a", StatusNotInstalled),
	}
	m.listScroll.cursor = 0

	ops := m.buildUninstallOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 uninstall ops for not-installed plugin, got %d", len(ops))
	}
}

func TestDestructiveOpsRejectRootSymlinkBeforeScheduling(t *testing.T) {
	rootLink := filepath.Join(t.TempDir(), "plugins")
	if err := os.Symlink(string(filepath.Separator), rootLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		build func(*Model) []pendingOp
	}{
		{name: "remove", build: (*Model).buildRemoveOps},
		{name: "uninstall", build: (*Model).buildUninstallOps},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil)
			m.cfg.PluginPath = mustRoot(t, rootLink)
			m.plugins = []PluginItem{testPluginItem("repo", "user/repo", StatusInstalled)}

			if ops := tt.build(&m); len(ops) != 0 {
				t.Fatalf("scheduled %d destructive operations through root symlink", len(ops))
			}
			if m.plugins[0].Status != StatusLoadFailed {
				t.Fatalf("status = %v, want %v", m.plugins[0].Status, StatusLoadFailed)
			}
		})
	}
}

func TestDestructiveOpsResolveRootImmediatelyBeforeRemoval(t *testing.T) {
	tests := []struct {
		name  string
		build func(*Model) []pendingOp
		run   func(pendingOp) tea.Cmd
		ok    func(tea.Msg) bool
	}{
		{
			name:  "remove",
			build: (*Model).buildRemoveOps,
			run:   removePluginDirCmd,
			ok: func(msg tea.Msg) bool {
				result, isResult := msg.(pluginRemoveResultMsg)
				return isResult && result.Success
			},
		},
		{
			name:  "uninstall",
			build: (*Model).buildUninstallOps,
			run:   uninstallPluginCmd,
			ok: func(msg tea.Msg) bool {
				result, isResult := msg.(pluginUninstallResultMsg)
				return isResult && result.Success
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			pluginRoot := filepath.Join(base, "real-plugins")
			pluginDir := filepath.Join(pluginRoot, "repo")
			if err := os.MkdirAll(pluginDir, 0o755); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(pluginDir, "keep")
			if err := os.WriteFile(marker, []byte("safe"), 0o600); err != nil {
				t.Fatal(err)
			}
			rootLink := filepath.Join(base, "plugins")
			if err := os.Symlink(pluginRoot, rootLink); err != nil {
				t.Fatal(err)
			}

			m := newTestModel(t, nil)
			m.cfg.PluginPath = mustRoot(t, rootLink)
			m.plugins = []PluginItem{testPluginItem("repo", "user/repo", StatusInstalled)}
			ops := tt.build(&m)
			if len(ops) != 1 {
				t.Fatalf("operations = %d, want 1", len(ops))
			}
			if err := os.Remove(rootLink); err != nil {
				t.Fatal(err)
			}

			if msg := tt.run(ops[0])(); tt.ok(msg) {
				t.Fatalf("destructive operation succeeded after plugin root became unresolved: %#v", msg)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("fixture changed after root resolution failure: %v", err)
			}
		})
	}
}

func TestBuildInstallOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusNotInstalled),
		testPluginItem("b", "user/b", StatusInstalled),
		testPluginItem("c", "user/c", StatusNotInstalled),
	}
	m.listScroll.cursor = 0

	ops := m.buildInstallOps()
	if len(ops) != 1 {
		t.Errorf("expected 1 install op (cursor on not-installed), got %d", len(ops))
	}
}

func TestBuildInstallOps_MultiSelect(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusNotInstalled),
		testPluginItem("b", "user/b", StatusInstalled),
		testPluginItem("c", "user/c", StatusNotInstalled),
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
		testPluginItem("a", "user/a", StatusInstalled),
		testPluginItem("b", "user/b", StatusInstalled),
	}
	m.listScroll.cursor = 0

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
		testPluginItem("a", "user/a", StatusNotInstalled),
		testPluginItem("b", "user/b", StatusInstalled),
		testPluginItem("c", "user/c", StatusNotInstalled),
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
		testPluginItem("a", "user/a", StatusInstalled),
	}

	ops := m.buildAutoInstallOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 auto install ops, got %d", len(ops))
	}
}

func TestBuildAutoUpdateOps(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusInstalled),
		testPluginItem("b", "user/b", StatusNotInstalled),
		testPluginItem("c", "user/c", StatusOutdated),
		testPluginItem("d", "user/d", StatusChecking),
	}

	ops := m.buildAutoUpdateOps()
	if len(ops) != 3 {
		t.Errorf("expected 3 auto update ops (installed+outdated+checking), got %d", len(ops))
	}
}

func TestBuildAutoUpdateOps_NoneInstalled(t *testing.T) {
	m := newTestModel(t, nil)
	m.plugins = []PluginItem{
		testPluginItem("a", "user/a", StatusNotInstalled),
	}

	ops := m.buildAutoUpdateOps()
	if len(ops) != 0 {
		t.Errorf("expected 0 auto update ops, got %d", len(ops))
	}
}

func TestDispatchNext_WithRunner_SourcesOnInstall(t *testing.T) {
	runner := tmux.NewMockRunner()
	cfg := &config.Config{PluginPath: mustRoot(t, t.TempDir()), TmuxConf: "/tmp/test.conf"}
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
	cfg := &config.Config{PluginPath: mustRoot(t, t.TempDir()), TmuxConf: "/tmp/test.conf"}
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
	cfg := &config.Config{PluginPath: mustRoot(t, t.TempDir()), TmuxConf: "/tmp/test.conf"}
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

func TestDispatchNext_BatchesUpToMax(t *testing.T) {
	m := newTestModel(t, nil)
	m.operation = OpInstall
	m.processing = true
	m.inFlight = 0
	m.pendingItems = make([]pendingOp, 5)
	for i := range m.pendingItems {
		m.pendingItems[i] = pendingOp{
			Name: fmt.Sprintf("plugin-%d", i),
			Spec: fmt.Sprintf("user/plugin-%d", i),
			Path: t.TempDir() + "/",
		}
	}

	cmd := m.dispatchNext()
	if cmd == nil {
		t.Fatal("expected non-nil command from dispatchNext")
	}
	if m.inFlight != maxConcurrentOps {
		t.Errorf("expected inFlight=%d, got %d", maxConcurrentOps, m.inFlight)
	}
	if len(m.pendingItems) != 5-maxConcurrentOps {
		t.Errorf("expected %d remaining pending, got %d", 5-maxConcurrentOps, len(m.pendingItems))
	}
	if len(m.inFlightNames) != maxConcurrentOps {
		t.Errorf("expected %d inFlightNames, got %d", maxConcurrentOps, len(m.inFlightNames))
	}
}

func TestDispatchNext_RespectsInFlightLimit(t *testing.T) {
	m := newTestModel(t, nil)
	m.operation = OpInstall
	m.processing = true
	m.inFlight = maxConcurrentOps
	m.pendingItems = []pendingOp{
		{Name: "extra", Spec: "user/extra", Path: t.TempDir() + "/"},
	}

	cmd := m.dispatchNext()
	if cmd != nil {
		t.Error("expected nil command when at concurrency limit")
	}
	if len(m.pendingItems) != 1 {
		t.Errorf("expected pending items unchanged, got %d", len(m.pendingItems))
	}
}

func TestHandleOpResult_DispatchesMore(t *testing.T) {
	m := newTestModel(t, nil)
	m.operation = OpInstall
	m.processing = true
	m.inFlight = maxConcurrentOps
	m.inFlightNames = make([]string, maxConcurrentOps)
	for i := range m.inFlightNames {
		m.inFlightNames[i] = fmt.Sprintf("inflight-%d", i)
	}
	m.totalItems = maxConcurrentOps + 2
	m.completedItems = 0
	m.pendingItems = []pendingOp{
		{Name: "next-a", Spec: "user/next-a", Path: t.TempDir() + "/"},
		{Name: "next-b", Spec: "user/next-b", Path: t.TempDir() + "/"},
	}

	result := ResultItem{Name: m.inFlightNames[0], Success: true, Message: "installed"}
	cmd := m.handleOpResult(result, nil)

	if m.inFlight != maxConcurrentOps {
		t.Errorf("expected inFlight to refill to %d, got %d", maxConcurrentOps, m.inFlight)
	}
	if cmd == nil {
		t.Error("expected non-nil command to dispatch next batch")
	}
	if m.completedItems != 1 {
		t.Errorf("expected completedItems=1, got %d", m.completedItems)
	}
	// The completed item should be removed from inFlightNames.
	for _, name := range m.inFlightNames {
		if name == result.Name {
			t.Errorf("expected %q removed from inFlightNames", result.Name)
		}
	}
}
