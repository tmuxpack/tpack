package plug_test

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func TestLegacyPluginName(t *testing.T) {
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
			if got := plug.LegacyPluginName(tt.raw); got != tt.want {
				t.Errorf("LegacyPluginName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRootChildUsesExactComponent(t *testing.T) {
	root, err := plug.NewRoot("test", "/tmp/plugins", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := root.Child("repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/plugins/repo.git" {
		t.Errorf("Child() = %q, want /tmp/plugins/repo.git", got)
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
		{"user/repo", "user/repo", "user/repo", "", ""},
		{"user/repo#develop", "user/repo", "user/repo", "develop", ""},
		{"https://github.com/user/plugin.git#main", "user/plugin", "https://github.com/user/plugin.git", "main", ""},
		{"simple", "simple", "simple", "", ""},
		{"catppuccin/tmux alias=catppuccin-tmux", "catppuccin/tmux", "catppuccin/tmux", "", "catppuccin-tmux"},
		{"catppuccin/tmux alias=catppuccin-tmux#v2", "catppuccin/tmux", "catppuccin/tmux", "v2", "catppuccin-tmux"},
		{"https://github.com/user/repo.git alias=my-plugin", "user/repo", "https://github.com/user/repo.git", "", "my-plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			p := mustParsePlugin(t, tt.raw)
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

func TestParseSpecUsesRepositoryDisplayName(t *testing.T) {
	tests := map[string]string{
		"catppuccin/tmux":                             "catppuccin/tmux",
		"catppuccin/tmux alias=theme":                 "catppuccin/tmux",
		"https://gitlab.com/group/subgroup/theme.git": "gitlab.com/group/subgroup/theme",
	}
	for raw, want := range tests {
		p, err := plug.ParseSpec(raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != want {
			t.Errorf("ParseSpec(%q).Name = %q, want %q", raw, p.Name, want)
		}
	}
}

func TestParseSpecBuildsIdentityMetadata(t *testing.T) {
	p, err := plug.ParseSpec("catppuccin/tmux#v2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Spec != "catppuccin/tmux" || p.Branch != "v2" {
		t.Fatalf("parsed spec = %#v", p)
	}
	if p.Identity != "github.com/catppuccin/tmux" {
		t.Errorf("Identity = %q", p.Identity)
	}
	if p.DirName != "tmux-87a1216f1f68" {
		t.Errorf("DirName = %q", p.DirName)
	}
}

func TestParseSpecAliasControlsOnlyDirectoryMetadata(t *testing.T) {
	p, err := plug.ParseSpec("catppuccin/tmux alias=catppuccin-theme#v2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != "github.com/catppuccin/tmux" || p.DirName != "catppuccin-theme" {
		t.Fatalf("parsed alias = %#v", p)
	}
}

func TestParseSpecWarnsOnExtraTokens(t *testing.T) {
	var warnings []string
	p, err := plug.ParseSpec("user/repo extra junk", func(msg string) { warnings = append(warnings, msg) })
	if err != nil {
		t.Fatal(err)
	}

	if p.Name != "user/repo" {
		t.Errorf("Name = %q, want %q", p.Name, "user/repo")
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
	p := mustParsePlugin(t, "user/repo extra")
	if p.Name != "user/repo" {
		t.Errorf("Name = %q, want %q", p.Name, "user/repo")
	}
}

func mustParsePlugin(t *testing.T, raw string) plug.Plugin {
	t.Helper()
	p, err := plug.ParseSpec(raw, nil)
	if err != nil {
		t.Fatalf("ParseSpec(%q): %v", raw, err)
	}
	return p
}
