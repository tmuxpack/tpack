package registry

import (
	"testing"
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
