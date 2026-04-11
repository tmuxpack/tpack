package registry

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseRegistry(t *testing.T) {
	raw := []byte(`
categories:
  - theme
  - session

plugins:
  - repo: catppuccin/tmux
    description: Soothing pastel theme for Tmux
    author: catppuccin
    category: theme
    stars: 1250
  - repo: tmux-plugins/tmux-resurrect
    description: Persists tmux environment across system restarts
    author: tmux-plugins
    category: session
    stars: 11400
`)
	reg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(reg.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(reg.Categories))
	}
	if len(reg.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(reg.Plugins))
	}
	p := reg.Plugins[0]
	if p.Repo != "catppuccin/tmux" {
		t.Errorf("expected repo catppuccin/tmux, got %s", p.Repo)
	}
	if p.Stars != 1250 {
		t.Errorf("expected 1250 stars, got %d", p.Stars)
	}
	if p.Category != "theme" {
		t.Errorf("expected category theme, got %s", p.Category)
	}
}

func TestParseRegistry_Empty(t *testing.T) {
	raw := []byte("categories: []\nplugins: []\n")
	reg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(reg.Plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(reg.Plugins))
	}
}

func TestParseRegistry_InvalidYAML(t *testing.T) {
	raw := []byte("{{invalid yaml")
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSearch(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "catppuccin/tmux", Description: "Soothing pastel theme", Category: "theme", Stars: 1250},
			{Repo: "tmux-plugins/tmux-resurrect", Description: "Persists tmux environment", Category: "session", Stars: 11400},
			{Repo: "tmux-plugins/tmux-sensible", Description: "Sensible defaults", Category: "utility", Stars: 5000},
		},
	}

	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"exact repo match", "catppuccin", 1},
		{"description match", "persists", 1},
		{"partial match", "tmux", 3},
		{"case insensitive", "SENSIBLE", 1},
		{"no match", "nonexistent", 0},
		{"empty query returns all", "", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Search(reg, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("Search(%q): got %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

func TestFilterByCategory(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "catppuccin/tmux", Category: "theme"},
			{Repo: "dracula/tmux", Category: "theme"},
			{Repo: "tmux-plugins/tmux-resurrect", Category: "session"},
		},
	}

	themes := FilterByCategory(reg, "theme")
	if len(themes) != 2 {
		t.Errorf("expected 2 themes, got %d", len(themes))
	}

	sessions := FilterByCategory(reg, "session")
	if len(sessions) != 1 {
		t.Errorf("expected 1 session plugin, got %d", len(sessions))
	}

	empty := FilterByCategory(reg, "navigation")
	if len(empty) != 0 {
		t.Errorf("expected 0 navigation plugins, got %d", len(empty))
	}
}

func TestParseRegistry_WithHost(t *testing.T) {
	raw := []byte(`
categories:
  - theme

plugins:
  - repo: catppuccin/tmux
    description: Soothing pastel theme for Tmux
    author: catppuccin
    category: theme
    stars: 1250
  - repo: gitlab-user/tmux-theme
    description: A GitLab-hosted theme
    author: gitlab-user
    category: theme
    stars: 42
    host: gitlab.com
`)
	reg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(reg.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(reg.Plugins))
	}
	if reg.Plugins[0].Host != "" {
		t.Errorf("expected empty host for GitHub plugin, got %q", reg.Plugins[0].Host)
	}
	if reg.Plugins[1].Host != "gitlab.com" {
		t.Errorf("expected host gitlab.com, got %q", reg.Plugins[1].Host)
	}
}

func TestParseRegistry_HostOmittedInOutput(t *testing.T) {
	item := RegistryItem{
		Repo:        "user/repo",
		Description: "test",
		Author:      "user",
		Category:    "theme",
		Stars:       0,
	}
	// Host is empty, should be omitted from YAML output
	data, err := yaml.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "host") {
		t.Errorf("expected host to be omitted from YAML when empty, got:\n%s", data)
	}

	// Host is set, should appear
	item.Host = "gitlab.com"
	data, err = yaml.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), "host: gitlab.com") {
		t.Errorf("expected host: gitlab.com in YAML output, got:\n%s", data)
	}
}

func TestNewest(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "old/plugin", Description: "Old plugin", Stars: 5000, AddedDate: "2026-01-01"},
			{Repo: "new/plugin", Description: "New plugin", Stars: 100, AddedDate: "2026-04-01"},
			{Repo: "mid/plugin", Description: "Mid plugin", Stars: 3000, AddedDate: "2026-02-15"},
			{Repo: "also-new/plugin", Description: "Also new", Stars: 500, AddedDate: "2026-04-01"},
		},
	}

	results := Newest(reg, 3)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Newest date first, then by stars within same date
	if results[0].Repo != "also-new/plugin" {
		t.Errorf("expected also-new/plugin first (same date, higher stars), got %s", results[0].Repo)
	}
	if results[1].Repo != "new/plugin" {
		t.Errorf("expected new/plugin second (same date, lower stars), got %s", results[1].Repo)
	}
	if results[2].Repo != "mid/plugin" {
		t.Errorf("expected mid/plugin third, got %s", results[2].Repo)
	}
}

func TestNewest_EmptyDatesSortLast(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "no-date/plugin", Description: "No date", Stars: 9999},
			{Repo: "has-date/plugin", Description: "Has date", Stars: 1, AddedDate: "2026-01-01"},
		},
	}

	results := Newest(reg, 2)
	if results[0].Repo != "has-date/plugin" {
		t.Errorf("expected has-date/plugin first, got %s", results[0].Repo)
	}
	if results[1].Repo != "no-date/plugin" {
		t.Errorf("expected no-date/plugin last, got %s", results[1].Repo)
	}
}

func TestNewest_NLargerThanPlugins(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "only/one", Description: "Only one", Stars: 10, AddedDate: "2026-03-01"},
		},
	}

	results := Newest(reg, 20)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestNewest_EmptyRegistry(t *testing.T) {
	reg := &Registry{}
	results := Newest(reg, 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_SortedByStars(t *testing.T) {
	reg := &Registry{
		Plugins: []RegistryItem{
			{Repo: "low-stars/tmux", Description: "tmux plugin", Stars: 10},
			{Repo: "high-stars/tmux", Description: "tmux plugin", Stars: 5000},
			{Repo: "mid-stars/tmux", Description: "tmux plugin", Stars: 500},
		},
	}

	results := Search(reg, "tmux")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Stars < results[1].Stars || results[1].Stars < results[2].Stars {
		t.Errorf("results not sorted by stars desc: %d, %d, %d",
			results[0].Stars, results[1].Stars, results[2].Stars)
	}
}
