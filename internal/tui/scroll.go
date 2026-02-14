package tui

// scrollState tracks cursor position and scroll offset for a scrollable list.
type scrollState struct {
	cursor       int
	scrollOffset int
}

// moveUp moves the cursor up and adjusts scroll.
func (s *scrollState) moveUp() {
	if s.cursor > 0 {
		s.cursor--
		if s.cursor < s.scrollOffset+ScrollOffsetMargin && s.scrollOffset > 0 {
			s.scrollOffset--
		}
	}
}

// moveDown moves the cursor down within a list of the given length,
// adjusting scroll based on the visible height.
func (s *scrollState) moveDown(listLen, visibleHeight int) {
	if s.cursor < listLen-1 {
		s.cursor++
		if s.cursor >= s.scrollOffset+visibleHeight-ScrollOffsetMargin {
			maxOffset := listLen - visibleHeight
			if maxOffset < 0 {
				maxOffset = 0
			}
			if s.scrollOffset < maxOffset {
				s.scrollOffset++
			}
		}
	}
}

// reset resets cursor and scroll to zero.
func (s *scrollState) reset() {
	s.cursor = 0
	s.scrollOffset = 0
}
