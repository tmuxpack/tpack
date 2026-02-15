package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/state"
)

func TestLoadMissingFile(t *testing.T) {
	s := state.Load(filepath.Join(t.TempDir(), "nonexistent"))
	if !s.LastUpdateCheck.IsZero() {
		t.Errorf("expected zero time, got %v", s.LastUpdateCheck)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "tpm")
	os.MkdirAll(statePath, 0o755)
	os.WriteFile(filepath.Join(statePath, "state.yml"), []byte("{{bad yaml!"), 0o644)

	s := state.Load(statePath)
	if !s.LastUpdateCheck.IsZero() {
		t.Errorf("expected zero time on corrupt file, got %v", s.LastUpdateCheck)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "tpm")

	now := time.Now().Truncate(time.Second)
	s := state.State{LastUpdateCheck: now}

	if err := state.Save(statePath, s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded := state.Load(statePath)
	if !loaded.LastUpdateCheck.Equal(now) {
		t.Errorf("LastUpdateCheck = %v, want %v", loaded.LastUpdateCheck, now)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "nested", "tpm")

	s := state.State{LastUpdateCheck: time.Now()}

	if err := state.Save(statePath, s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(statePath, "state.yml")); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}

func TestSaveAndLoadSelfUpdateCheck(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "tpm")

	now := time.Now().Truncate(time.Second)
	s := state.State{LastSelfUpdateCheck: now}

	if err := state.Save(statePath, s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded := state.Load(statePath)
	if !loaded.LastSelfUpdateCheck.Equal(now) {
		t.Errorf("LastSelfUpdateCheck = %v, want %v", loaded.LastSelfUpdateCheck, now)
	}
}

func TestLoadExistingStateWithoutSelfUpdateCheck(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "tpm")
	os.MkdirAll(statePath, 0o755)

	// Write a state file with only the old field (backward compat).
	content := "last_update_check: 2026-01-01T00:00:00Z\n"
	os.WriteFile(filepath.Join(statePath, "state.yml"), []byte(content), 0o644)

	loaded := state.Load(statePath)
	if loaded.LastUpdateCheck.IsZero() {
		t.Error("expected LastUpdateCheck to be set")
	}
	if !loaded.LastSelfUpdateCheck.IsZero() {
		t.Errorf("expected LastSelfUpdateCheck to be zero, got %v", loaded.LastSelfUpdateCheck)
	}
}

func TestSaveOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "tpm")

	first := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	second := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	state.Save(statePath, state.State{LastUpdateCheck: first})
	state.Save(statePath, state.State{LastUpdateCheck: second})

	loaded := state.Load(statePath)
	if !loaded.LastUpdateCheck.Equal(second) {
		t.Errorf("LastUpdateCheck = %v, want %v", loaded.LastUpdateCheck, second)
	}
}
