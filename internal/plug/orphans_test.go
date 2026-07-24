package plug_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func mkPluginDirs(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.Mkdir(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFindOrphansDetectsUndeclaredDir(t *testing.T) {
	dir := mkPluginDirs(t, "tmux-sensible", "tmux-yank")
	declared := []plug.Plugin{mustParsePlugin(t, "tmux-plugins/tmux-sensible")}

	orphans, err := plug.FindOrphans(declared, mustRoot(t, dir))
	if err != nil {
		t.Fatal(err)
	}

	if len(orphans) != 1 {
		t.Fatalf("orphans = %v, want exactly one", orphans)
	}
	if orphans[0].Name != "tmux-yank" {
		t.Errorf("orphan name = %q, want tmux-yank", orphans[0].Name)
	}
}

func TestFindOrphansIgnoresTpmAndDeclared(t *testing.T) {
	dir := mkPluginDirs(t, "tmux-sensible", "tpm", "tpack")
	declared := []plug.Plugin{mustParsePlugin(t, "tmux-plugins/tmux-sensible")}

	orphans, err := plug.FindOrphans(declared, mustRoot(t, dir))
	if err != nil {
		t.Fatal(err)
	}

	if len(orphans) != 0 {
		t.Errorf("orphans = %v, want none", orphans)
	}
}

func TestFindOrphansEmptyPluginListReportsAll(t *testing.T) {
	// A readable config with zero @plugin lines legitimately means "no
	// plugins declared" -- clean should report every installed directory
	// (except tpm/tpack) as an orphan, matching TPM's original contract.
	// A missing/unreadable config is handled upstream as an explicit error,
	// not by this function.
	dir := mkPluginDirs(t, "tmux-sensible", "tmux-yank", "tmux-resurrect")

	orphans, err := plug.FindOrphans(nil, mustRoot(t, dir))
	if err != nil {
		t.Fatal(err)
	}

	if len(orphans) != 3 {
		t.Fatalf("orphans = %v, want all 3 dirs reported for empty plugin list", orphans)
	}
}

func mustRoot(t *testing.T, path string) plug.Root {
	t.Helper()
	root, err := plug.NewRoot("test", path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
