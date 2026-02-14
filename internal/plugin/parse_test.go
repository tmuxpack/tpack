package plugin_test

import (
	"testing"

	"github.com/tmux-plugins/tpm/internal/plugin"
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
			got := plugin.ExtractPluginsFromConfig(tt.content)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plugin.ExtractSourcedFiles(tt.content)
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

func TestManualExpansion(t *testing.T) {
	home := "/home/user"
	tests := []struct {
		path string
		want string
	}{
		{"~/foo", "/home/user/foo"},
		{"$HOME/foo", "/home/user/foo"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", "/home/user"},
		{"$HOME", "/home/user"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := plugin.ManualExpansion(tt.path, home)
			if got != tt.want {
				t.Errorf("ManualExpansion(%q, %q) = %q, want %q", tt.path, home, got, tt.want)
			}
		})
	}
}
