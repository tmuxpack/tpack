package plug_test

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func TestPluginName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"user/repo", "repo"},
		{"https://github.com/user/plugin.git", "plugin"},
		{"git@github.com:user/plugin.git", "plugin"},
		{"tmux-plugins/tmux-sensible", "tmux-sensible"},
		{"https://git::@github.com/user/tmux-yank", "tmux-yank"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := plug.PluginName(tt.raw)
			if got != tt.want {
				t.Errorf("PluginName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRootChildUsesPluginName(t *testing.T) {
	root, err := plug.NewRoot("test", "/tmp/plugins", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := root.Child("https://github.com/user/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/plugins/repo" {
		t.Errorf("Child() = %q, want /tmp/plugins/repo", got)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user/repo", "https://git::@github.com/user/repo"},
		{"https://github.com/user/repo.git", "https://github.com/user/repo.git"},
		{"git@github.com:user/repo.git", "git@github.com:user/repo.git"},
		{"https://git::@github.com/user/repo", "https://git::@github.com/user/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := plug.NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		raw    string
		name   string
		spec   string
		branch string
		alias  string
	}{
		{"user/repo", "repo", "user/repo", "", ""},
		{"user/repo#develop", "repo", "user/repo", "develop", ""},
		{"https://github.com/user/plugin.git#main", "plugin", "https://github.com/user/plugin.git", "main", ""},
		{"simple", "simple", "simple", "", ""},
		{"catppuccin/tmux alias=catppuccin-tmux", "catppuccin-tmux", "catppuccin/tmux", "", "catppuccin-tmux"},
		{"catppuccin/tmux alias=catppuccin-tmux#v2", "catppuccin-tmux", "catppuccin/tmux", "v2", "catppuccin-tmux"},
		{"https://github.com/user/repo.git alias=my-plugin", "my-plugin", "https://github.com/user/repo.git", "", "my-plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			p := plug.ParseSpec(tt.raw, nil)
			if p.Name != tt.name {
				t.Errorf("Name = %q, want %q", p.Name, tt.name)
			}
			if p.Spec != tt.spec {
				t.Errorf("Spec = %q, want %q", p.Spec, tt.spec)
			}
			if p.Branch != tt.branch {
				t.Errorf("Branch = %q, want %q", p.Branch, tt.branch)
			}
			if p.Alias != tt.alias {
				t.Errorf("Alias = %q, want %q", p.Alias, tt.alias)
			}
		})
	}
}

func TestParseSpecWarnsOnExtraTokens(t *testing.T) {
	var warnings []string
	p := plug.ParseSpec("user/repo extra junk", func(msg string) { warnings = append(warnings, msg) })

	if p.Name != "repo" {
		t.Errorf("Name = %q, want %q", p.Name, "repo")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	want := `plugin spec "user/repo extra junk" has unexpected extra tokens: extra junk`
	if warnings[0] != want {
		t.Errorf("warning = %q, want %q", warnings[0], want)
	}
}

func TestParseSpecNilWarnDoesNotPanic(t *testing.T) {
	p := plug.ParseSpec("user/repo extra", nil)
	if p.Name != "repo" {
		t.Errorf("Name = %q, want %q", p.Name, "repo")
	}
}
