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
