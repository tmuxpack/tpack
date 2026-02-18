package cli_test

import (
	"path/filepath"
	"testing"

	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
)

func TestValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git CLI test in short mode")
	}

	bare := initBareRepo(t)
	clone := cloneLocal(t, bare)

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{
			name: "valid git repo",
			dir:  clone,
			want: true,
		},
		{
			name: "non-git directory",
			dir:  t.TempDir(),
			want: false,
		},
		{
			name: "nonexistent directory",
			dir:  filepath.Join(t.TempDir(), "does-not-exist"),
			want: false,
		},
	}

	v := gitcli.NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.IsGitRepo(tt.dir)
			if got != tt.want {
				t.Errorf("IsGitRepo(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestValidator_RealDir(t *testing.T) {
	v := gitcli.NewValidator()
	// /tmp is not a git repo
	if v.IsGitRepo("/tmp") {
		t.Error("/tmp should not be a git repo")
	}
}
