package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestRenderCursor(t *testing.T) {
	active := renderCursor(true)
	if active != "> " {
		t.Errorf("renderCursor(true) = %q, want %q", active, "> ")
	}
	inactive := renderCursor(false)
	if inactive != "  " {
		t.Errorf("renderCursor(false) = %q, want %q", inactive, "  ")
	}
}

func TestRenderCheckbox(t *testing.T) {
	th := DefaultTheme()
	checked := th.renderCheckbox(true)
	if !strings.Contains(checked, "\u2713") {
		t.Errorf("renderCheckbox(true) should contain checkmark, got %q", checked)
	}
	unchecked := th.renderCheckbox(false)
	if !strings.Contains(unchecked, "[ ]") {
		t.Errorf("renderCheckbox(false) should contain '[ ]', got %q", unchecked)
	}
}

func TestRenderScrollIndicators(t *testing.T) {
	th := DefaultTheme()
	// Both indicators visible — data range shrinks by one on each side.
	top, bottom, ds, de := th.renderScrollIndicators(2, 8, 10)
	if top == "" {
		t.Error("expected top indicator when start > 0")
	}
	if bottom == "" {
		t.Error("expected bottom indicator when end < total")
	}
	if ds != 3 {
		t.Errorf("expected dataStart=3, got %d", ds)
	}
	if de != 7 {
		t.Errorf("expected dataEnd=7, got %d", de)
	}

	// Only top — data range shrinks at start only.
	top, bottom, ds, de = th.renderScrollIndicators(2, 10, 10)
	if top == "" {
		t.Error("expected top indicator when start > 0")
	}
	if bottom != "" {
		t.Error("expected no bottom indicator when end == total")
	}
	if ds != 3 {
		t.Errorf("expected dataStart=3, got %d", ds)
	}
	if de != 10 {
		t.Errorf("expected dataEnd=10, got %d", de)
	}

	// Only bottom — data range shrinks at end only.
	top, bottom, ds, de = th.renderScrollIndicators(0, 5, 10)
	if top != "" {
		t.Error("expected no top indicator when start == 0")
	}
	if bottom == "" {
		t.Error("expected bottom indicator when end < total")
	}
	if ds != 0 {
		t.Errorf("expected dataStart=0, got %d", ds)
	}
	if de != 4 {
		t.Errorf("expected dataEnd=4, got %d", de)
	}

	// Neither — data range unchanged.
	top, bottom, ds, de = th.renderScrollIndicators(0, 10, 10)
	if top != "" {
		t.Error("expected no top indicator")
	}
	if bottom != "" {
		t.Error("expected no bottom indicator")
	}
	if ds != 0 {
		t.Errorf("expected dataStart=0, got %d", ds)
	}
	if de != 10 {
		t.Errorf("expected dataEnd=10, got %d", de)
	}
}

func TestCalculateVisibleRange(t *testing.T) {
	// Normal range.
	start, end := calculateVisibleRange(0, 5, 10)
	if start != 0 || end != 5 {
		t.Errorf("expected (0, 5), got (%d, %d)", start, end)
	}

	// End clamped to total.
	start, end = calculateVisibleRange(8, 5, 10)
	if start != 8 || end != 10 {
		t.Errorf("expected (8, 10), got (%d, %d)", start, end)
	}

	// Zero items.
	start, end = calculateVisibleRange(0, 5, 0)
	if start != 0 || end != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", start, end)
	}
}

func testBinding(k, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc))
}

func TestRenderHelp(t *testing.T) {
	th := DefaultTheme()
	// Single binding produces non-empty output.
	out := th.renderHelp(80, testBinding("q", "quit"))
	if out == "" {
		t.Error("expected non-empty help output")
	}
	if !strings.Contains(out, "quit") {
		t.Errorf("expected help to contain 'quit', got %q", out)
	}

	// Large width keeps everything on one line (no wrapping).
	out = th.renderHelp(200, testBinding("q", "quit"), testBinding("i", "install"))
	lines := strings.Split(out, "\n")
	// The help is rendered with lipgloss which may add margin, but the core
	// content should stay on a single logical line when width is very large.
	if len(lines) > 2 {
		t.Errorf("expected at most 2 lines (content + possible margin), got %d", len(lines))
	}
}

func TestRenderHelpInlineHint(t *testing.T) {
	th := DefaultTheme()
	styledQ := th.HelpKeyStyle.Render("q")

	t.Run("single char key highlights inline in description", func(t *testing.T) {
		out := th.renderHelp(80, testBinding("i", "install"))
		styledI := th.HelpKeyStyle.Render("i")
		// Should contain the highlighted "i" followed by "nstall" (inline),
		// not "i install" (separate key + desc).
		if !strings.Contains(out, styledI+"nstall") {
			t.Errorf("expected inline highlight %q+\"nstall\" in %q", styledI, out)
		}
	})

	t.Run("case insensitive match highlights key case", func(t *testing.T) {
		out := th.renderHelp(80, testBinding("d", "Delete"))
		styledD := th.HelpKeyStyle.Render("d")
		// "d" should match "D" in "Delete", but render as lowercase "d".
		if !strings.Contains(out, styledD+"elete") {
			t.Errorf("expected inline highlight %q+\"elete\" in %q", styledD, out)
		}
	})

	t.Run("multi char key renders separately", func(t *testing.T) {
		out := th.renderHelp(80, testBinding("esc", "back"))
		styledEsc := th.HelpKeyStyle.Render("esc")
		if !strings.Contains(out, styledEsc+" back") {
			t.Errorf("expected separate %q+\" back\" in %q", styledEsc, out)
		}
	})

	t.Run("single char not in desc renders separately", func(t *testing.T) {
		out := th.renderHelp(80, testBinding("x", "close"))
		styledX := th.HelpKeyStyle.Render("x")
		if !strings.Contains(out, styledX+" close") {
			t.Errorf("expected separate %q+\" close\" in %q", styledX, out)
		}
	})

	t.Run("q quit still highlights inline", func(t *testing.T) {
		out := th.renderHelp(80, testBinding("q", "quit"))
		if !strings.Contains(out, styledQ+"uit") {
			t.Errorf("expected inline highlight %q+\"uit\" in %q", styledQ, out)
		}
	})
}
