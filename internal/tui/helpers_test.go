package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

func mustRoot(t *testing.T, path string) plug.Root {
	t.Helper()
	root, err := plug.NewRoot("test", path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestBuildPluginItems_AllNotInstalled(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()
	plugins := []plug.Plugin{
		testPlugin("tmux-sensible", "tmux-plugins/tmux-sensible"),
		testPlugin("tmux-yank", "tmux-plugins/tmux-yank"),
	}

	items := buildPluginItems(plugins, mustRoot(t, pluginPath), validator, nil)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, item := range items {
		if item.Status != StatusNotInstalled {
			t.Errorf("expected StatusNotInstalled for %s, got %s", item.Name, item.Status)
		}
	}
}

func TestBuildPluginItems_Installed(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()

	// Create plugin directory.
	dir := filepath.Join(pluginPath, "tmux-sensible")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Mark as valid git repo.
	validator.Valid[dir] = true

	plugins := []plug.Plugin{
		testPlugin("tmux-sensible", "tmux-plugins/tmux-sensible"),
	}

	items := buildPluginItems(plugins, mustRoot(t, pluginPath), validator, nil)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != StatusChecking {
		t.Errorf("expected StatusChecking, got %s", items[0].Status)
	}
}

func TestBuildPluginItems_PreservesFields(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()
	plugin := mustParsePlugin(t, "tmux-plugins/tmux-yank#main")
	dir := filepath.Join(pluginPath, plugin.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	validator.Valid[dir] = true

	items := buildPluginItems([]plug.Plugin{plugin}, mustRoot(t, pluginPath), validator, nil)

	if items[0].Raw != "tmux-plugins/tmux-yank#main" {
		t.Errorf("expected raw spec preserved, got %q", items[0].Raw)
	}
	if items[0].Name != "tmux-yank" {
		t.Errorf("expected name tmux-yank, got %s", items[0].Name)
	}
	if items[0].Identity != "github.com/tmux-plugins/tmux-yank" {
		t.Errorf("expected normalized identity, got %q", items[0].Identity)
	}
	if items[0].DirName != "tmux-yank-d3dd4adebd91" {
		t.Errorf("expected canonical directory, got %q", items[0].DirName)
	}
	if items[0].Spec != "tmux-plugins/tmux-yank" {
		t.Errorf("expected spec tmux-plugins/tmux-yank, got %s", items[0].Spec)
	}
	if items[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", items[0].Branch)
	}
	if items[0].Status != StatusChecking {
		t.Errorf("expected DirName directory to be detected, got %s", items[0].Status)
	}
}

func TestLoadErrorMapUsesDirectoryKeyWithLegacyNameFallback(t *testing.T) {
	got := loadErrorMap([]plug.LoadFailure{
		{Name: "tmux", DirName: "tmux--f0e8a426", Message: "new record"},
		{Name: "legacy", Message: "old record"},
	})
	if got.byDirName["tmux--f0e8a426"] != "new record" {
		t.Errorf("directory-keyed error = %q", got.byDirName["tmux--f0e8a426"])
	}
	if got.byLegacyName["legacy"] != "old record" {
		t.Errorf("legacy name-keyed error = %q", got.byLegacyName["legacy"])
	}
}

func TestBuildPluginItemsMatchesLegacyLoadErrorByName(t *testing.T) {
	pluginPath := t.TempDir()
	plugin := mustParsePlugin(t, "catppuccin/tmux")
	dir := filepath.Join(pluginPath, plugin.DirName)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	validator := git.NewMockValidator()
	validator.Valid[dir] = true
	loadErrors := loadErrorMap([]plug.LoadFailure{{Name: "tmux", Message: "legacy failure"}})

	items := buildPluginItems([]plug.Plugin{plugin}, mustRoot(t, pluginPath), validator, loadErrors)

	if items[0].Status != StatusLoadFailed || items[0].LoadErr != "legacy failure" {
		t.Fatalf("legacy load failure was not correlated: %+v", items[0])
	}
}

func TestBuildPluginItemsDoesNotTreatDirectoryKeyAsLegacyName(t *testing.T) {
	pluginPath := t.TempDir()
	plugins := []plug.Plugin{
		{Name: "source", DirName: "shared-key", Spec: "owner/source"},
		{Name: "shared-key", DirName: "other-directory", Spec: "owner/other"},
	}
	validator := git.NewMockValidator()
	for _, plugin := range plugins {
		dir := filepath.Join(pluginPath, plugin.DirName)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		validator.Valid[dir] = true
	}
	loadErrors := loadErrorMap([]plug.LoadFailure{{
		Name: "source", DirName: "shared-key", Message: "source failed",
	}})

	items := buildPluginItems(plugins, mustRoot(t, pluginPath), validator, loadErrors)

	if items[0].Status != StatusLoadFailed || items[0].LoadErr != "source failed" {
		t.Fatalf("directory-matched plugin = %+v", items[0])
	}
	if items[1].Status != StatusChecking || items[1].LoadErr != "" {
		t.Fatalf("unrelated plugin inherited directory-keyed failure: %+v", items[1])
	}
}

func TestFindOrphans_NoOrphans(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "tmux-sensible", DirName: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	// Create only the listed plugin directory.
	os.MkdirAll(filepath.Join(pluginPath, "tmux-sensible"), 0o755)

	orphans, err := findOrphans(plugins, mustRoot(t, pluginPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestFindOrphans_WithOrphans(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "tmux-sensible", DirName: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	// Create listed plugin + orphan directory.
	os.MkdirAll(filepath.Join(pluginPath, "tmux-sensible"), 0o755)
	os.MkdirAll(filepath.Join(pluginPath, "tmux-old"), 0o755)

	orphans, err := findOrphans(plugins, mustRoot(t, pluginPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].Name != "tmux-old" {
		t.Errorf("expected orphan name tmux-old, got %s", orphans[0].Name)
	}
}

func TestFindOrphans_SkipsTpm(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "declared", Spec: "user/declared"},
	}

	// tpm directory should always be skipped.
	os.MkdirAll(filepath.Join(pluginPath, "tpm"), 0o755)
	os.MkdirAll(filepath.Join(pluginPath, "orphan"), 0o755)

	orphans, err := findOrphans(plugins, mustRoot(t, pluginPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan (tpm skipped), got %d", len(orphans))
	}
	if orphans[0].Name != "orphan" {
		t.Errorf("expected orphan name orphan, got %s", orphans[0].Name)
	}
}

func TestFindOrphans_EmptyDir(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "test", Spec: "user/test"},
	}

	orphans, err := findOrphans(plugins, mustRoot(t, pluginPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans for empty dir, got %d", len(orphans))
	}
}

func TestFindOrphansResolvesRootBeforeEnumeration(t *testing.T) {
	realTarget := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Symlink(realTarget, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(target, "orphan"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "plugins")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	orphans, err := findOrphans(nil, mustRoot(t, link))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 {
		t.Fatalf("orphans = %v, want one", orphans)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resolvedTarget, "orphan")
	if orphans[0].Path != want {
		t.Errorf("orphan path = %q, want resolved path %q", orphans[0].Path, want)
	}
}

func TestBuildPluginItems_LoadFailed(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()

	dir := filepath.Join(pluginPath, "tmux-statusline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	validator.Valid[dir] = true

	plugins := []plug.Plugin{testPlugin("tmux-statusline", "x/tmux-statusline")}
	loadErrors := loadErrorMap([]plug.LoadFailure{{
		Name: "tmux-statusline", DirName: "tmux-statusline", Message: "exec format error",
	}})

	items := buildPluginItems(plugins, mustRoot(t, pluginPath), validator, loadErrors)

	if items[0].Status != StatusLoadFailed {
		t.Errorf("expected StatusLoadFailed, got %s", items[0].Status)
	}
	if items[0].LoadErr != "exec format error" {
		t.Errorf("expected LoadErr set, got %q", items[0].LoadErr)
	}
}

func TestBuildPluginItems_LoadErrorIgnoredWhenNotInstalled(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()

	plugins := []plug.Plugin{testPlugin("ghost", "x/ghost")}
	loadErrors := loadErrorMap([]plug.LoadFailure{{
		Name: "ghost", DirName: "ghost", Message: "stale error",
	}})

	items := buildPluginItems(plugins, mustRoot(t, pluginPath), validator, loadErrors)

	if items[0].Status != StatusNotInstalled {
		t.Errorf("expected StatusNotInstalled for uninstalled plugin, got %s", items[0].Status)
	}
	if items[0].LoadErr != "" {
		t.Errorf("expected no LoadErr for uninstalled plugin, got %q", items[0].LoadErr)
	}
}

func TestLoadFailedStatusStrings(t *testing.T) {
	if StatusLoadFailed.String() != "Loading Failed" {
		t.Errorf("expected 'Loading Failed', got %q", StatusLoadFailed.String())
	}
	if !StatusLoadFailed.IsInstalled() {
		t.Error("expected StatusLoadFailed to count as installed")
	}
}
