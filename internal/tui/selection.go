package tui

// toggleSelection toggles the selection state for the given index.
func (m *Model) toggleSelection(idx int) {
	if m.selected[idx] {
		delete(m.selected, idx)
	} else {
		m.selected[idx] = true
	}
	m.multiSelectActive = len(m.selected) > 0
}

// clearSelection removes all selections.
func (m *Model) clearSelection() {
	m.selected = make(map[int]bool)
	m.multiSelectActive = false
}

// selectionCount returns how many items are selected.
func (m *Model) selectionCount() int {
	return len(m.selected)
}

// selectedIndices returns the indices of selected items in order.
func (m *Model) selectedIndices() []int {
	indices := make([]int, 0, len(m.selected))
	for i := 0; i < len(m.plugins); i++ {
		if m.selected[i] {
			indices = append(indices, i)
		}
	}
	return indices
}
