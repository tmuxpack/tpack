package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plug"
)

func TestBuildPluginItems_AllNotInstalled(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	validator := git.NewMockValidator()
	plugins := []plug.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
		{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank"},
	}

	items := buildPluginItems(plugins, pluginPath, validator)

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
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	items := buildPluginItems(plugins, pluginPath, validator)

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
	plugins := []plug.Plugin{
		{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Branch: "main"},
	}

	items := buildPluginItems(plugins, pluginPath, validator)

	if items[0].Name != "tmux-yank" {
		t.Errorf("expected name tmux-yank, got %s", items[0].Name)
	}
	if items[0].Spec != "tmux-plugins/tmux-yank" {
		t.Errorf("expected spec tmux-plugins/tmux-yank, got %s", items[0].Spec)
	}
	if items[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", items[0].Branch)
	}
}

func TestFindOrphans_NoOrphans(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	// Create only the listed plugin directory.
	os.MkdirAll(filepath.Join(pluginPath, "tmux-sensible"), 0o755)

	orphans := findOrphans(plugins, pluginPath)
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestFindOrphans_WithOrphans(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{
		{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	// Create listed plugin + orphan directory.
	os.MkdirAll(filepath.Join(pluginPath, "tmux-sensible"), 0o755)
	os.MkdirAll(filepath.Join(pluginPath, "tmux-old"), 0o755)

	orphans := findOrphans(plugins, pluginPath)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].Name != "tmux-old" {
		t.Errorf("expected orphan name tmux-old, got %s", orphans[0].Name)
	}
}

func TestFindOrphans_SkipsTpm(t *testing.T) {
	pluginPath := t.TempDir() + "/"
	plugins := []plug.Plugin{}

	// tpm directory should always be skipped.
	os.MkdirAll(filepath.Join(pluginPath, "tpm"), 0o755)
	os.MkdirAll(filepath.Join(pluginPath, "orphan"), 0o755)

	orphans := findOrphans(plugins, pluginPath)
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

	orphans := findOrphans(plugins, pluginPath)
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans for empty dir, got %d", len(orphans))
	}
}
