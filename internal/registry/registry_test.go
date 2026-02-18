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
