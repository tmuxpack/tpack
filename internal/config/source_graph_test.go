package config_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
)

func TestLoadSourceGraphRecursesInOrder(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source ~/.tmux/one.conf"
	fs.Files["/home/user/.tmux/one.conf"] = "source ~/.tmux/two.conf"
	fs.Files["/home/user/.tmux/two.conf"] = "set -g @plugin owner/deep"
	graph, err := config.LoadSourceGraph(fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/.tmux/one.conf", "/home/user/.tmux/two.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphRequiredFailure(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source ~/.tmux/missing.conf"
	_, err := config.LoadSourceGraph(fs, testPaths(t))
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
	if sourceErr.Parent != "/home/user/.tmux.conf" || sourceErr.Target != "/home/user/.tmux/missing.conf" {
		t.Errorf("error context = %#v", sourceErr)
	}
}

func TestLoadSourceGraphOptionalFailure(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -q ~/.tmux/missing.conf"
	graph, err := config.LoadSourceGraph(fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.Paths(); !reflect.DeepEqual(got, []string{"/home/user/.tmux.conf"}) {
		t.Errorf("paths = %v", got)
	}
}

func TestLoadSourceGraphDeduplicatesCycles(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source ~/.tmux/a.conf"
	fs.Files["/home/user/.tmux/a.conf"] = "source ~/.tmux/b.conf"
	fs.Files["/home/user/.tmux/b.conf"] = "source ~/.tmux/a.conf"
	graph, err := config.LoadSourceGraph(fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/.tmux/a.conf", "/home/user/.tmux/b.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphOrdersEtcBeforeUser(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = "set -g @plugin owner/system"
	fs.Files["/home/user/.tmux.conf"] = "set -g @plugin owner/user"
	graph, err := config.LoadSourceGraph(fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/etc/tmux.conf", "/home/user/.tmux.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphExpandsXDGPath(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source $XDG_CONFIG_HOME/tmux/plugins.conf"
	fs.Files["/home/user/.config/tmux/plugins.conf"] = "set -g @plugin owner/xdg"
	graph, err := config.LoadSourceGraph(fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/.config/tmux/plugins.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}
