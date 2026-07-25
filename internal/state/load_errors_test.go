package state_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/state"
)

func TestSaveAndLoadLoadErrors(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "tpack")
	want := []plug.LoadFailure{
		{Name: "tmux-statusline", DirName: "tmux-statusline-15a02faf769b", Message: "error sourcing statusline.tmux: exec format error"},
		{Name: "other", Message: "boom"},
	}

	if err := state.SaveLoadErrors(statePath, want); err != nil {
		t.Fatalf("SaveLoadErrors failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(statePath, "load-errors.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "dir_name:") != 1 {
		t.Errorf("persisted YAML must contain one non-empty dir_name:\n%s", data)
	}

	got := state.LoadLoadErrors(statePath, nil)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("round-trip mismatch:\n got %v\nwant %v", got, want)
	}
}

func TestLoadLoadErrorsReadsLegacyRecordsWithoutDirName(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "tpack")
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := "load_errors:\n  - name: tmux-statusline\n    message: legacy failure\n"
	if err := os.WriteFile(filepath.Join(statePath, "load-errors.yml"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	got := state.LoadLoadErrors(statePath, nil)
	if len(got) != 1 {
		t.Fatalf("legacy failures = %v", got)
	}
	if got[0].Name != "tmux-statusline" || got[0].DirName != "" || got[0].Message != "legacy failure" {
		t.Errorf("legacy failure = %#v", got[0])
	}
}

func TestSaveLoadErrorsOverwrites(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "tpack")
	_ = state.SaveLoadErrors(statePath, []plug.LoadFailure{{Name: "a", Message: "1"}})
	_ = state.SaveLoadErrors(statePath, []plug.LoadFailure{{Name: "b", Message: "2"}})

	got := state.LoadLoadErrors(statePath, nil)
	if len(got) != 1 || got[0].Name != "b" {
		t.Errorf("expected only the second save to remain, got %v", got)
	}
}

func TestSaveLoadErrorsEmptyRemovesFile(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "tpack")
	_ = state.SaveLoadErrors(statePath, []plug.LoadFailure{{Name: "a", Message: "1"}})

	if err := state.SaveLoadErrors(statePath, nil); err != nil {
		t.Fatalf("SaveLoadErrors(nil) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(statePath, "load-errors.yml")); !os.IsNotExist(err) {
		t.Error("expected load-errors.yml to be removed on empty save")
	}
	if got := state.LoadLoadErrors(statePath, nil); got != nil {
		t.Errorf("expected nil after empty save, got %v", got)
	}
}

func TestLoadLoadErrorsCorruptFile(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "tpack")
	os.MkdirAll(statePath, 0o755)
	os.WriteFile(filepath.Join(statePath, "load-errors.yml"), []byte("{{bad yaml!"), 0o644)

	if got := state.LoadLoadErrors(statePath, nil); got != nil {
		t.Errorf("expected nil on corrupt file, got %v", got)
	}
}

func TestLoadLoadErrorsCorruptFileWarns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "load-errors.yml"), []byte("{not yaml: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	var warnings []string
	failures := state.LoadLoadErrors(dir, func(msg string) { warnings = append(warnings, msg) })

	if failures != nil {
		t.Errorf("corrupt file must yield nil failures, got %v", failures)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "corrupt load-errors file") {
		t.Errorf("warning = %q, want it to contain %q", warnings[0], "corrupt load-errors file")
	}
}

func TestLoadLoadErrorsCorruptFileNilWarn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "load-errors.yml"), []byte("{not yaml: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = state.LoadLoadErrors(dir, nil) // must not panic
}

func TestLoadLoadErrorsMissingFile(t *testing.T) {
	if got := state.LoadLoadErrors(filepath.Join(t.TempDir(), "tpack"), nil); got != nil {
		t.Errorf("expected nil for missing file, got %v", got)
	}
}
