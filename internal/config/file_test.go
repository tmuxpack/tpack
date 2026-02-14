package config_test

import (
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestResolveLoadsColors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want config.ColorConfig
	}{
		{
			name: "full config",
			yaml: `colors:
  primary: "#111111"
  secondary: "#222222"
  accent: "#333333"
  error: "#444444"
  muted: "#555555"
  text: "#666666"
`,
			want: config.ColorConfig{
				Primary:   "#111111",
				Secondary: "#222222",
				Accent:    "#333333",
				Error:     "#444444",
				Muted:     "#555555",
				Text:      "#666666",
			},
		},
		{
			name: "partial config",
			yaml: `colors:
  primary: "#aabbcc"
  text: "#ddeeff"
`,
			want: config.ColorConfig{
				Primary: "#aabbcc",
				Text:    "#ddeeff",
			},
		},
		{
			name: "empty file",
			yaml: "",
			want: config.ColorConfig{},
		},
		{
			name: "malformed yaml",
			yaml: "{{bad yaml!",
			want: config.ColorConfig{},
		},
		{
			name: "no colors key",
			yaml: "something_else: true\n",
			want: config.ColorConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			fs := config.NewMockFS()
			fs.Files["/home/user/.config/tpm/config.yml"] = tt.yaml

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Colors != tt.want {
				t.Errorf("Colors = %+v, want %+v", cfg.Colors, tt.want)
			}
		})
	}
}

func TestResolveNoConfigFile(t *testing.T) {
	m := tmux.NewMockRunner()
	fs := config.NewMockFS()
	// No config file in fs.

	cfg, err := config.Resolve(m, testOpts(fs)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Colors != (config.ColorConfig{}) {
		t.Errorf("Colors = %+v, want zero value", cfg.Colors)
	}
}

func TestResolveLoadsUpdates(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantInterval time.Duration
		wantMode     string
	}{
		{
			name: "full update config",
			yaml: `updates:
  check_interval: "24h"
  mode: "prompt"
`,
			wantInterval: 24 * time.Hour,
			wantMode:     "prompt",
		},
		{
			name: "auto mode with 1h interval",
			yaml: `updates:
  check_interval: "1h"
  mode: "auto"
`,
			wantInterval: 1 * time.Hour,
			wantMode:     "auto",
		},
		{
			name: "off mode",
			yaml: `updates:
  mode: "off"
`,
			wantInterval: 0,
			wantMode:     "off",
		},
		{
			name:         "no updates section",
			yaml:         "colors:\n  primary: \"#aabbcc\"\n",
			wantInterval: 0,
			wantMode:     "",
		},
		{
			name:         "empty file",
			yaml:         "",
			wantInterval: 0,
			wantMode:     "",
		},
		{
			name: "invalid interval string",
			yaml: `updates:
  check_interval: "not-a-duration"
  mode: "prompt"
`,
			wantInterval: 0,
			wantMode:     "prompt",
		},
		{
			name: "zero interval string",
			yaml: `updates:
  check_interval: "0"
  mode: "auto"
`,
			wantInterval: 0,
			wantMode:     "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			fs := config.NewMockFS()
			fs.Files["/home/user/.config/tpm/config.yml"] = tt.yaml

			cfg, err := config.Resolve(m, testOpts(fs)...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.UpdateCheckInterval != tt.wantInterval {
				t.Errorf("UpdateCheckInterval = %v, want %v", cfg.UpdateCheckInterval, tt.wantInterval)
			}
			if cfg.UpdateMode != tt.wantMode {
				t.Errorf("UpdateMode = %q, want %q", cfg.UpdateMode, tt.wantMode)
			}
		})
	}
}
