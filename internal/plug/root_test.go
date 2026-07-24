package plug_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
)

func TestNewRoot(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "absolute", raw: "/home/user/plugins", want: "/home/user/plugins"},
		{name: "cleans", raw: "/home/user/../user/plugins/.", want: "/home/user/plugins"},
		{name: "trims tmux value", raw: "  /home/user/plugins  ", want: "/home/user/plugins"},
		{name: "expands home", raw: "~/plugins", want: "/home/user/plugins"},
		{name: "empty", raw: "", wantErr: true},
		{name: "whitespace", raw: "  ", wantErr: true},
		{name: "relative dot", raw: ".", wantErr: true},
		{name: "relative parent", raw: "../plugins", wantErr: true},
		{name: "root", raw: string(filepath.Separator), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := plug.NewRoot("test", tt.raw, "/home/user", "/home/user/.config")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewRoot(%q) returned no error", tt.raw)
				}
				var rootErr *plug.RootError
				if !errors.As(err, &rootErr) {
					t.Fatalf("error type = %T, want *plug.RootError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRoot(%q): %v", tt.raw, err)
			}
			path, err := root.Path()
			if err != nil {
				t.Fatal(err)
			}
			if path != tt.want {
				t.Errorf("Path() = %q, want %q", path, tt.want)
			}
			if !strings.HasSuffix(root.String(), string(filepath.Separator)) {
				t.Errorf("String() = %q, want trailing separator", root.String())
			}
		})
	}
}

func TestRootChildPreservesExactComponent(t *testing.T) {
	root, err := plug.NewRoot("test", "/home/user/plugins", "", "")
	if err != nil {
		t.Fatal(err)
	}

	got, err := root.Child("repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/user/plugins/repo.git" {
		t.Errorf("Child() = %q", got)
	}

	for _, raw := range []string{"", ".", "..", "/", "owner/repo.git", `owner\repo.git`} {
		if _, err := root.Child(raw); err == nil {
			t.Errorf("Child(%q) returned no error", raw)
		}
	}
}

func TestZeroRootRejected(t *testing.T) {
	var root plug.Root
	if _, err := root.Path(); err == nil {
		t.Fatal("zero Root.Path returned no error")
	}
	if _, err := root.Child("repo"); err == nil {
		t.Fatal("zero Root.Child returned no error")
	}
}

func TestRootResolved(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "plugins")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	root, err := plug.NewRoot("test", link, "", "")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := root.Resolved()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := resolved.Path()
	want, _ := filepath.EvalSymlinks(target)
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
}

func TestRootResolvedRejectsFilesystemRoot(t *testing.T) {
	link := filepath.Join(t.TempDir(), "plugins")
	if err := os.Symlink(string(filepath.Separator), link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	root, err := plug.NewRoot("test", link, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.Resolved(); err == nil {
		t.Fatal("Resolved accepted a symlink to the filesystem root")
	}
}
