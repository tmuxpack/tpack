package plug_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func TestNormalizeIdentity(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want string
	}{
		{name: "github shorthand", spec: "catppuccin/tmux", want: "github.com/catppuccin/tmux"},
		{name: "https", spec: "https://github.com/catppuccin/tmux.git", want: "github.com/catppuccin/tmux"},
		{name: "https credentials", spec: "https://git::@github.com/catppuccin/tmux", want: "github.com/catppuccin/tmux"},
		{name: "ssh URL", spec: "ssh://git@github.com/catppuccin/tmux.git", want: "github.com/catppuccin/tmux"},
		{name: "scp", spec: "git@github.com:catppuccin/tmux.git", want: "github.com/catppuccin/tmux"},
		{name: "scp without user", spec: "github.com:catppuccin/tmux.git", want: "github.com/catppuccin/tmux"},
		{name: "host normalized", spec: "https://GitLab.COM/group/tmux.git", want: "gitlab.com/group/tmux"},
		{name: "non-default port retained", spec: "ssh://git@example.com:2222/team/tmux.git", want: "example.com:2222/team/tmux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plug.NormalizeIdentity(tt.spec)
			if err != nil {
				t.Fatalf("NormalizeIdentity(%q): %v", tt.spec, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeIdentity(%q) = %q, want %q", tt.spec, got, tt.want)
			}
		})
	}
}

func TestNormalizeIdentityRejectsUnsafeRepositorySpecs(t *testing.T) {
	tests := []string{
		`https://github.com/owner/repo";run-shell`,
		"https://github.com/owner/repo\nset -g @plugin attacker/repo",
		"https://github.com/owner/../repo",
		"https://github.com/owner/./repo",
		"https://github.com/owner//repo",
		`https://github.com;run-shell/owner/repo`,
		`https://attacker";@github.com/owner/repo`,
		`github.com;run-shell:owner/repo`,
	}

	for _, spec := range tests {
		t.Run(spec, func(t *testing.T) {
			if identity, err := plug.NormalizeIdentity(spec); err == nil {
				t.Fatalf("NormalizeIdentity(%q) = %q, want error", spec, identity)
			}
		})
	}
}

func TestNormalizeIdentityKeepsOwnersAndHostsDistinct(t *testing.T) {
	inputs := []string{
		"github.com/catppuccin/tmux",
		"github.com/dracula/tmux",
		"gitlab.com/catppuccin/tmux",
	}
	seen := make(map[string]bool)
	for _, input := range inputs {
		identity, err := plug.NormalizeIdentity("https://" + input)
		if err != nil {
			t.Fatal(err)
		}
		if seen[identity] {
			t.Fatalf("duplicate identity %q", identity)
		}
		seen[identity] = true
	}
}

func TestRepositoryName(t *testing.T) {
	tests := map[string]string{
		"github.com/catppuccin/tmux":        "catppuccin/tmux",
		"gitlab.com/group/subgroup/project": "gitlab.com/group/subgroup/project",
	}
	for identity, want := range tests {
		if got := plug.RepositoryName(identity); got != want {
			t.Errorf("RepositoryName(%q) = %q, want %q", identity, got, want)
		}
	}
}

func TestGeneratedDirNameIsDistinctDeterministicAndBounded(t *testing.T) {
	if got := plug.GeneratedDirName("github.com/catppuccin/tmux"); got != "tmux-87a1216f1f68" {
		t.Errorf("catppuccin directory = %q", got)
	}
	if got := plug.GeneratedDirName("github.com/dracula/tmux"); got != "tmux-e74ab6318c07" {
		t.Errorf("dracula directory = %q", got)
	}
	longIdentity := "github.com/owner/" + strings.Repeat("very-long-repository-", 8)
	got := plug.GeneratedDirName(longIdentity)
	if len(got) > 64 {
		t.Fatalf("generated directory has %d bytes: %q", len(got), got)
	}
	if !regexp.MustCompile(`^[a-z0-9._-]+-[0-9a-f]{12}$`).MatchString(got) {
		t.Fatalf("unsafe generated directory %q", got)
	}
}
