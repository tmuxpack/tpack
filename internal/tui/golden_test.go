package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

// ansiRe matches all ANSI SGR escape sequences (colors, bold, italic, etc.)
// and OSC sequences used by lipgloss for hyperlinks.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\].*?\x1b\\|\x1b\][^\x1b]*\x07`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	stripped := stripANSI(got)
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(stripped), 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s not found (run with -update to create): %v", path, err)
	}
	if stripped != string(expected) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s\n--- diff lines ---\n%s",
			name, string(expected), stripped, diffLines(string(expected), stripped))
	}
}

func diffLines(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	var b strings.Builder
	total := len(wantLines)
	if len(gotLines) > total {
		total = len(gotLines)
	}
	for i := 0; i < total; i++ {
		wl, gl := "", ""
		if i < len(wantLines) {
			wl = wantLines[i]
		}
		if i < len(gotLines) {
			gl = gotLines[i]
		}
		if wl != gl {
			b.WriteString(fmt.Sprintf("line %d:\n  want: %q\n  got:  %q\n", i+1, wl, gl))
		}
	}
	return b.String()
}

func TestStripANSI(t *testing.T) {
	input := "\x1b[1m\x1b[38;5;99mhello\x1b[0m world"
	got := stripANSI(input)
	want := "hello world"
	if got != want {
		t.Errorf("stripANSI: got %q, want %q", got, want)
	}
}

func TestGolden_ScreenList(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "list_empty",
			setup: func(m *Model) {
				m.plugins = nil
			},
		},
		{
			name: "list_single_plugin",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusNotInstalled},
				}
			},
		},
		{
			name: "list_few_plugins",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusInstalled},
					{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Status: StatusNotInstalled},
					{Name: "tmux-resurrect", Spec: "tmux-plugins/tmux-resurrect", Status: StatusOutdated},
				}
			},
		},
		{
			name: "list_fill_screen",
			setup: func(m *Model) {
				// viewHeight = FixedHeight - TitleReservedLines = 25 - 12 = 13
				m.plugins = make([]PluginItem, 13)
				for i := range m.plugins {
					m.plugins[i] = PluginItem{
						Name:   fmt.Sprintf("plugin-%02d", i+1),
						Spec:   fmt.Sprintf("user/plugin-%02d", i+1),
						Status: StatusInstalled,
					}
				}
			},
		},
		{
			name: "list_overflow",
			setup: func(m *Model) {
				// 20 plugins > viewHeight (13) → scroll indicators appear.
				m.plugins = make([]PluginItem, 20)
				for i := range m.plugins {
					m.plugins[i] = PluginItem{
						Name:   fmt.Sprintf("plugin-%02d", i+1),
						Spec:   fmt.Sprintf("user/plugin-%02d", i+1),
						Status: StatusInstalled,
					}
				}
			},
		},
		{
			name: "list_long_names",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-very-long-plugin-name-that-stretches", Spec: "user/long1", Status: StatusInstalled},
					{Name: "another-extremely-long-name-plugin", Spec: "user/long2", Status: StatusNotInstalled},
					{Name: "short", Spec: "user/short", Status: StatusOutdated},
				}
			},
		},
		{
			name: "list_with_orphans",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusInstalled},
					{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Status: StatusInstalled},
				}
				m.orphans = []OrphanItem{
					{Name: "old-plugin", Path: "/tmp/old-plugin"},
					{Name: "stale-plugin", Path: "/tmp/stale-plugin"},
				}
			},
		},
		{
			name: "list_multiselect",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusInstalled},
					{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Status: StatusNotInstalled},
					{Name: "tmux-resurrect", Spec: "tmux-plugins/tmux-resurrect", Status: StatusInstalled},
				}
				m.multiSelectActive = true
				m.selected = map[int]bool{0: true, 2: true}
			},
		},
		{
			name: "list_all_installed",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusInstalled},
					{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Status: StatusInstalled},
					{Name: "tmux-resurrect", Spec: "tmux-plugins/tmux-resurrect", Status: StatusInstalled},
				}
			},
		},
		{
			name: "list_all_not_installed",
			setup: func(m *Model) {
				m.plugins = []PluginItem{
					{Name: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible", Status: StatusNotInstalled},
					{Name: "tmux-yank", Spec: "tmux-plugins/tmux-yank", Status: StatusNotInstalled},
					{Name: "tmux-resurrect", Spec: "tmux-plugins/tmux-resurrect", Status: StatusNotInstalled},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil)
			tt.setup(&m)
			assertGolden(t, tt.name, m.View())
		})
	}
}
