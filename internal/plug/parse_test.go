package plug_test

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func TestExtractPluginsFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "set -g @plugin with double quotes",
			content: `set -g @plugin "tmux-plugins/tmux-sensible"`,
			want:    []string{"tmux-plugins/tmux-sensible"},
		},
		{
			name:    "set-option -g @plugin with single quotes",
			content: `set-option -g @plugin 'tmux-plugins/tmux-yank'`,
			want:    []string{"tmux-plugins/tmux-yank"},
		},
		{
			name:    "leading whitespace",
			content: `    set -g @plugin "tmux-plugins/tmux-sensible"`,
			want:    []string{"tmux-plugins/tmux-sensible"},
		},
		{
			name:    "tab leading whitespace",
			content: "\tset -g @plugin 'tmux-plugins/tmux-sensible'",
			want:    []string{"tmux-plugins/tmux-sensible"},
		},
		{
			name:    "comments ignored",
			content: "# set -g @plugin \"tmux-plugins/tmux-sensible\"",
			want:    nil,
		},
		{
			name:    "non-plugin set lines ignored",
			content: `set -g status-right ""`,
			want:    nil,
		},
		{
			name: "multiple plugins",
			content: `set -g @plugin "tmux-plugins/tpm"
set -g @plugin "tmux-plugins/tmux-sensible"
set -g @plugin "tmux-plugins/tmux-yank"`,
			want: []string{
				"tmux-plugins/tpm",
				"tmux-plugins/tmux-sensible",
				"tmux-plugins/tmux-yank",
			},
		},
		{
			name:    "no quotes",
			content: `set -g @plugin tmux-plugins/tmux-sensible`,
			want:    []string{"tmux-plugins/tmux-sensible"},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "full URL with quotes",
			content: `set -g @plugin "https://github.com/user/plugin.git"`,
			want:    []string{"https://github.com/user/plugin.git"},
		},
		{
			name:    "double-quoted value containing single quote",
			content: `set -g @plugin "foo'bar"`,
			want:    []string{"foo'bar"},
		},
		{
			name:    "single-quoted value containing double quote",
			content: `set -g @plugin 'foo"bar'`,
			want:    []string{`foo"bar`},
		},
		{
			name:    "double-quoted value with special chars",
			content: `set -g @plugin "user/plug-in_v2.0"`,
			want:    []string{"user/plug-in_v2.0"},
		},
		{
			name:    "single-quoted full URL",
			content: `set -g @plugin 'https://github.com/user/plugin.git'`,
			want:    []string{"https://github.com/user/plugin.git"},
		},
		{
			name:    "unquoted plugin with trailing whitespace",
			content: "set -g @plugin tmux-plugins/tpm   ",
			want:    []string{"tmux-plugins/tpm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plug.ExtractPluginsFromConfig(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d plugins %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("plugin[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractSourcedFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "source with path",
			content: `source ~/.tmux/theme.conf`,
			want:    []string{"~/.tmux/theme.conf"},
		},
		{
			name:    "source-file with path",
			content: `source-file ~/.tmux/theme.conf`,
			want:    []string{"~/.tmux/theme.conf"},
		},
		{
			name:    "source-file with -q flag",
			content: `source-file -q ~/.tmux/local.conf`,
			want:    []string{"~/.tmux/local.conf"},
		},
		{
			name:    "quoted path",
			content: `source-file "~/.tmux/theme.conf"`,
			want:    []string{"~/.tmux/theme.conf"},
		},
		{
			name:    "single quoted path",
			content: `source-file '~/.tmux/theme.conf'`,
			want:    []string{"~/.tmux/theme.conf"},
		},
		{
			name:    "comment ignored",
			content: `# source ~/.tmux/theme.conf`,
			want:    nil,
		},
		{
			name:    "unquoted path with inline comment",
			content: `source ~/.tmux/theme.conf # my theme`,
			want:    []string{"~/.tmux/theme.conf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plug.ExtractSourcedFiles(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d files %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("file[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchesPluginLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		spec string
		want bool
	}{
		{
			name: "double-quoted match",
			line: `set -g @plugin "tmux-plugins/tmux-sensible"`,
			spec: "tmux-plugins/tmux-sensible",
			want: true,
		},
		{
			name: "single-quoted match",
			line: `set -g @plugin 'tmux-plugins/tmux-sensible'`,
			spec: "tmux-plugins/tmux-sensible",
			want: true,
		},
		{
			name: "unquoted match",
			line: `set -g @plugin tmux-plugins/tmux-sensible`,
			spec: "tmux-plugins/tmux-sensible",
			want: true,
		},
		{
			name: "set-option match",
			line: `set-option -g @plugin "tmux-plugins/tmux-sensible"`,
			spec: "tmux-plugins/tmux-sensible",
			want: true,
		},
		{
			name: "leading whitespace match",
			line: `    set -g @plugin "tmux-plugins/tmux-sensible"`,
			spec: "tmux-plugins/tmux-sensible",
			want: true,
		},
		{
			name: "different spec no match",
			line: `set -g @plugin "tmux-plugins/tmux-yank"`,
			spec: "tmux-plugins/tmux-sensible",
			want: false,
		},
		{
			name: "comment no match",
			line: `# set -g @plugin "tmux-plugins/tmux-sensible"`,
			spec: "tmux-plugins/tmux-sensible",
			want: false,
		},
		{
			name: "non-plugin line no match",
			line: `set -g status-right ""`,
			spec: "tmux-plugins/tmux-sensible",
			want: false,
		},
		{
			name: "empty line no match",
			line: "",
			spec: "tmux-plugins/tmux-sensible",
			want: false,
		},
		{
			name: "partial spec no match",
			line: `set -g @plugin "tmux-plugins/tmux-sensible-extra"`,
			spec: "tmux-plugins/tmux-sensible",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plug.MatchesPluginLine(tt.line, tt.spec)
			if got != tt.want {
				t.Errorf("MatchesPluginLine(%q, %q) = %v, want %v", tt.line, tt.spec, got, tt.want)
			}
		})
	}
}

func TestManualExpansion(t *testing.T) {
	home := "/home/user"
	xdg := "/home/user/.config"
	tests := []struct {
		name string
		path string
		xdg  string
		want string
	}{
		{"tilde prefix", "~/foo", xdg, "/home/user/foo"},
		{"$HOME prefix", "$HOME/foo", xdg, "/home/user/foo"},
		{"${HOME} prefix", "${HOME}/foo", xdg, "/home/user/foo"},
		{"absolute path", "/absolute/path", xdg, "/absolute/path"},
		{"relative path", "relative/path", xdg, "relative/path"},
		{"bare tilde", "~", xdg, "/home/user"},
		{"bare $HOME", "$HOME", xdg, "/home/user"},
		{"bare ${HOME}", "${HOME}", xdg, "/home/user"},
		{"$XDG_CONFIG_HOME prefix", "$XDG_CONFIG_HOME/tmux", xdg, "/home/user/.config/tmux"},
		{"${XDG_CONFIG_HOME} prefix", "${XDG_CONFIG_HOME}/tmux", xdg, "/home/user/.config/tmux"},
		{"bare $XDG_CONFIG_HOME", "$XDG_CONFIG_HOME", xdg, "/home/user/.config"},
		{"bare ${XDG_CONFIG_HOME}", "${XDG_CONFIG_HOME}", xdg, "/home/user/.config"},
		{"XDG empty falls through", "$XDG_CONFIG_HOME/tmux", "", "$XDG_CONFIG_HOME/tmux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plug.ManualExpansion(tt.path, home, tt.xdg)
			if got != tt.want {
				t.Errorf("ManualExpansion(%q, %q, %q) = %q, want %q", tt.path, home, tt.xdg, got, tt.want)
			}
		})
	}
}
