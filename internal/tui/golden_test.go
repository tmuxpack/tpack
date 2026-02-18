package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/registry"
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

func TestGolden_ScreenProgress(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "progress_in_flight",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpInstall
				m.totalItems = 3
				m.completedItems = 1
				m.processing = true
				m.inFlightNames = []string{"tmux-yank", "tmux-resurrect"}
			},
		},
		{
			name: "progress_complete_success",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpInstall
				m.totalItems = 3
				m.completedItems = 3
				m.processing = false
				m.results = []ResultItem{
					{Name: "tmux-sensible", Success: true, Message: "installed"},
					{Name: "tmux-yank", Success: true, Message: "installed"},
					{Name: "tmux-resurrect", Success: true, Message: "installed"},
				}
			},
		},
		{
			name: "progress_complete_mixed",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpInstall
				m.totalItems = 3
				m.completedItems = 3
				m.processing = false
				m.results = []ResultItem{
					{Name: "tmux-sensible", Success: true, Message: "installed"},
					{Name: "tmux-yank", Success: false, Message: "clone failed: timeout"},
					{Name: "tmux-resurrect", Success: true, Message: "installed"},
				}
			},
		},
		{
			name: "progress_complete_with_commits",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpUpdate
				m.totalItems = 3
				m.completedItems = 3
				m.processing = false
				m.results = []ResultItem{
					{Name: "tmux-sensible", Success: true, Message: "updated", Commits: []git.Commit{
						{Hash: "abc1234", Message: "add feature X"},
						{Hash: "def5678", Message: "fix bug Y"},
						{Hash: "ghi9012", Message: "refactor Z"},
					}},
					{Name: "tmux-yank", Success: true, Message: "updated", Commits: []git.Commit{
						{Hash: "jkl3456", Message: "bump version"},
					}},
					{Name: "tmux-resurrect", Success: true, Message: "already up-to-date"},
				}
			},
		},
		{
			name: "progress_auto_op",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpInstall
				m.autoOp = OpInstall
				m.totalItems = 2
				m.completedItems = 2
				m.processing = false
				m.results = []ResultItem{
					{Name: "tmux-sensible", Success: true, Message: "installed"},
					{Name: "tmux-yank", Success: true, Message: "installed"},
				}
			},
		},
		{
			name: "progress_many_results",
			setup: func(m *Model) {
				m.screen = ScreenProgress
				m.operation = OpInstall
				m.totalItems = 15
				m.completedItems = 15
				m.processing = false
				m.results = make([]ResultItem, 15)
				for i := range m.results {
					m.results[i] = ResultItem{
						Name:    fmt.Sprintf("plugin-%02d", i+1),
						Success: i%3 != 2, // every 3rd fails
						Message: "done",
					}
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

func TestGolden_ScreenCommits(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "commits_few",
			setup: func(m *Model) {
				m.screen = ScreenCommits
				m.commitViewName = "tmux-sensible"
				m.commitViewCommits = []git.Commit{
					{Hash: "abc1234", Message: "add feature X"},
					{Hash: "def5678", Message: "fix bug Y"},
					{Hash: "ghi9012", Message: "refactor Z"},
				}
			},
		},
		{
			name: "commits_overflow",
			setup: func(m *Model) {
				m.screen = ScreenCommits
				m.commitViewName = "tmux-yank"
				// commitViewerReservedLines = 13, so max visible = 25 - 13 = 12.
				// Use 20 commits to trigger scroll indicators.
				commits := make([]git.Commit, 20)
				for i := range commits {
					commits[i] = git.Commit{
						Hash:    fmt.Sprintf("%07x", i+1),
						Message: fmt.Sprintf("commit message %d", i+1),
					}
				}
				m.commitViewCommits = commits
			},
		},
		{
			name: "commits_single",
			setup: func(m *Model) {
				m.screen = ScreenCommits
				m.commitViewName = "tmux-resurrect"
				m.commitViewCommits = []git.Commit{
					{Hash: "abc1234", Message: "bump version to 1.0"},
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

func TestGolden_ScreenDebug(t *testing.T) {
	m := newTestModel(t, nil)
	m.screen = ScreenDebug
	m.version = "1.2.3"
	m.binaryPath = "/usr/local/bin/tpack"
	assertGolden(t, "debug_view", m.View())
}

func TestGolden_ScreenSearch(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "search_loading",
			setup: func(m *Model) {
				m.screen = ScreenSearch
				m.searchLoading = true
			},
		},
		{
			name: "search_empty_results",
			setup: func(m *Model) {
				m.screen = ScreenSearch
				m.searchRegistry = &registry.Registry{
					Categories: []string{"theme", "session"},
				}
				m.searchCategory = -1
			},
		},
		{
			name: "search_with_results",
			setup: func(m *Model) {
				m.screen = ScreenSearch
				m.searchRegistry = &registry.Registry{
					Categories: []string{"theme", "session"},
				}
				m.searchResults = []registry.RegistryItem{
					{Repo: "catppuccin/tmux", Description: "Soothing pastel theme", Category: "theme", Stars: 1250},
					{Repo: "tmux-plugins/tmux-resurrect", Description: "Persists tmux environment", Category: "session", Stars: 11400},
				}
				m.searchCategory = -1
			},
		},
		{
			name: "search_category_filter",
			setup: func(m *Model) {
				m.screen = ScreenSearch
				m.searchRegistry = &registry.Registry{
					Categories: []string{"theme", "session"},
				}
				m.searchResults = []registry.RegistryItem{
					{Repo: "catppuccin/tmux", Description: "Soothing pastel theme", Category: "theme", Stars: 1250},
				}
				m.searchCategory = 0
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
