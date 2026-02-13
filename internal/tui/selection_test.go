package tui

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/plugin"
)

func TestToggleSelection(t *testing.T) {
	m := newTestModel(t, []plugin.Plugin{
		{Name: "a", Spec: "user/a"},
		{Name: "b", Spec: "user/b"},
	})

	m.toggleSelection(0)
	if !m.selected[0] {
		t.Error("expected index 0 to be selected")
	}
	if !m.multiSelectActive {
		t.Error("expected multiSelectActive to be true")
	}

	// Toggle off.
	m.toggleSelection(0)
	if m.selected[0] {
		t.Error("expected index 0 to be deselected")
	}
	if m.multiSelectActive {
		t.Error("expected multiSelectActive to be false after deselecting all")
	}
}

func TestToggleSelection_Multiple(t *testing.T) {
	m := newTestModel(t, []plugin.Plugin{
		{Name: "a", Spec: "user/a"},
		{Name: "b", Spec: "user/b"},
		{Name: "c", Spec: "user/c"},
	})

	m.toggleSelection(0)
	m.toggleSelection(2)
	if len(m.selected) != 2 {
		t.Errorf("expected 2 selected, got %d", len(m.selected))
	}

	indices := m.selectedIndices()
	if len(indices) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 2 {
		t.Errorf("expected indices [0, 2], got %v", indices)
	}
}

func TestClearSelection(t *testing.T) {
	m := newTestModel(t, []plugin.Plugin{
		{Name: "a", Spec: "user/a"},
		{Name: "b", Spec: "user/b"},
	})

	m.toggleSelection(0)
	m.toggleSelection(1)
	m.clearSelection()

	if len(m.selected) != 0 {
		t.Errorf("expected 0 selected after clear, got %d", len(m.selected))
	}
	if m.multiSelectActive {
		t.Error("expected multiSelectActive to be false after clear")
	}
}

func TestSelection_Empty(t *testing.T) {
	m := newTestModel(t, nil)
	if len(m.selected) != 0 {
		t.Errorf("expected 0 selected, got %d", len(m.selected))
	}
}

func TestSelectedIndices_Order(t *testing.T) {
	m := newTestModel(t, []plugin.Plugin{
		{Name: "a", Spec: "user/a"},
		{Name: "b", Spec: "user/b"},
		{Name: "c", Spec: "user/c"},
		{Name: "d", Spec: "user/d"},
	})

	// Select in reverse order.
	m.toggleSelection(3)
	m.toggleSelection(1)

	indices := m.selectedIndices()
	if len(indices) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(indices))
	}
	// Should be in ascending order.
	if indices[0] != 1 || indices[1] != 3 {
		t.Errorf("expected indices [1, 3], got %v", indices)
	}
}
