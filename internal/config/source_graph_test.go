package config_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func TestLoadSourceGraphRecursesInOrder(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source ~/.tmux/one.conf"
	fs.Files["/home/user/.tmux/one.conf"] = "source ~/.tmux/two.conf"
	fs.Files["/home/user/.tmux/two.conf"] = "set -g @plugin owner/deep"
	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
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
	_, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
	if sourceErr.Parent != "/home/user/.tmux.conf" || sourceErr.Target != "/home/user/.tmux/missing.conf" {
		t.Errorf("error context = %#v", sourceErr)
	}
	want := "cannot read required source /home/user/.tmux/missing.conf from /home/user/.tmux.conf " +
		`(directive "source ~/.tmux/missing.conf"): file does not exist`
	if got := err.Error(); got != want {
		t.Fatalf("error text = %q, want %q", got, want)
	}
}

func TestLoadSourceGraphOptionalFailure(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -q ~/.tmux/missing.conf"
	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
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
	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
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
	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/etc/tmux.conf", "/home/user/.tmux.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphPreservesSuppliedRootOrderWhenEtcIsPresent(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/custom/first.conf"] = "set -g @plugin owner/custom"
	fs.Files["/etc/tmux.conf"] = "set -g @plugin owner/system"
	paths := testPaths(t)
	paths.TmuxConf = "/custom/first.conf"
	paths.TmuxConfs = []string{"/custom/first.conf", "/etc/tmux.conf"}

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, paths)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/custom/first.conf", "/etc/tmux.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want supplied order %v", got, want)
	}
}

func TestLoadSourceGraphExpandsXDGPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/home/user/.config")
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source $XDG_CONFIG_HOME/tmux/plugins.conf"
	fs.Files["/home/user/.config/tmux/plugins.conf"] = "set -g @plugin owner/xdg"
	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/.config/tmux/plugins.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphExpandsMultiplePathsInDirectiveOrder(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file ~/.tmux/one.conf ~/.tmux/two.conf"
	fs.Files["/home/user/.tmux/one.conf"] = "set -g @plugin owner/one"
	fs.Files["/home/user/.tmux/two.conf"] = "set -g @plugin owner/two"

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/home/user/.tmux.conf",
		"/home/user/.tmux/one.conf",
		"/home/user/.tmux/two.conf",
	}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphExpandsGlobInStableOrder(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file ~/.config/tmux/conf.d/*.conf"
	fs.Files["/home/user/.config/tmux/conf.d/20-second.conf"] = "set -g @plugin owner/two"
	fs.Files["/home/user/.config/tmux/conf.d/10-first.conf"] = "set -g @plugin owner/one"

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/home/user/.tmux.conf",
		"/home/user/.config/tmux/conf.d/10-first.conf",
		"/home/user/.config/tmux/conf.d/20-second.conf",
	}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphExpandsEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		tmuxEnv  string
		process  string
		wantPath string
	}{
		{
			name:     "tmux environment takes precedence",
			raw:      "$TPACK_SOURCE_DIR/extra.conf",
			tmuxEnv:  "/from-tmux",
			process:  "/from-process",
			wantPath: "/from-tmux/extra.conf",
		},
		{
			name:     "process environment is fallback",
			raw:      "${TPACK_SOURCE_DIR}/extra.conf",
			process:  "/from-process",
			wantPath: "/from-process/extra.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TPACK_SOURCE_DIR", tt.process)
			runner := tmux.NewMockRunner()
			if tt.tmuxEnv != "" {
				runner.Environment["TPACK_SOURCE_DIR"] = tt.tmuxEnv
			}
			fs := config.NewMockFS()
			fs.Files["/home/user/.tmux.conf"] = "source-file " + tt.raw
			fs.Files[tt.wantPath] = "set -g @plugin owner/expanded"

			graph, err := config.LoadSourceGraph(runner, fs, testPaths(t))
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"/home/user/.tmux.conf", tt.wantPath}
			if got := graph.Paths(); !reflect.DeepEqual(got, want) {
				t.Fatalf("paths = %v, want %v", got, want)
			}
		})
	}
}

func TestLoadSourceGraphExpandsSourceFormat(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{HOME}/dynamic.conf"] = "/home/user/dynamic.conf"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -F '#{HOME}/dynamic.conf'"
	fs.Files["/home/user/dynamic.conf"] = "set -g @plugin owner/dynamic"

	graph, err := config.LoadSourceGraph(runner, fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/dynamic.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphResolvesRelativePathsFromParent(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source .tmux/one.conf"
	fs.Files["/home/user/.tmux/one.conf"] = "source nested/two.conf"
	fs.Files["/home/user/.tmux/nested/two.conf"] = "set -g @plugin owner/two"

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/home/user/.tmux.conf",
		"/home/user/.tmux/one.conf",
		"/home/user/.tmux/nested/two.conf",
	}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphAllowsOptionalGlobMiss(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -q ~/.tmux/conf.d/*.conf"

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.Paths(); !reflect.DeepEqual(got, []string{"/home/user/.tmux.conf"}) {
		t.Fatalf("paths = %v, want only root config", got)
	}
}

func TestLoadSourceGraphValidatesParseOnlyDocumentWithoutExecutingNestedSources(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -n parse-only.conf\n"
	fs.Files["/home/user/parse-only.conf"] = "source-file missing-nested.conf\nset -g @plugin owner/ignored\n"

	graph, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/parse-only.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want parsed documents %v", got, want)
	}
}

func TestLoadSourceGraphRequiresParseOnlyTarget(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -n missing.conf\n"

	_, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
}

func TestLoadSourceGraphValidatesParseOnlyTargetSyntax(t *testing.T) {
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "source-file -n parse-only.conf\n"
	fs.Files["/home/user/parse-only.conf"] = "source-file 'unterminated\n"

	_, err := config.LoadSourceGraph(tmux.NewMockRunner(), fs, testPaths(t))
	var parseErr *config.ConfigParseError
	if !errors.As(err, &parseErr) || parseErr.Path != "/home/user/parse-only.conf" {
		t.Fatalf("error = %#v, want ConfigParseError for parse-only target", err)
	}
}

func TestLoadSourceGraphSkipsRequiredSourceInInactiveConditional(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{enabled}"] = "0"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "%if '#{enabled}'\nsource missing.conf\n%else\nsource active.conf\n%endif\n"
	fs.Files["/home/user/active.conf"] = "set -g @plugin owner/active"

	graph, err := config.LoadSourceGraph(runner, fs, testPaths(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.tmux.conf", "/home/user/active.conf"}
	if got := graph.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestLoadSourceGraphReturnsRequiredFailureFromActiveConditional(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{enabled}"] = "1"
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = "%if '#{enabled}'\nsource missing.conf\n%else\nsource active.conf\n%endif\n"
	fs.Files["/home/user/active.conf"] = "set -g @plugin owner/active"

	_, err := config.LoadSourceGraph(runner, fs, testPaths(t))
	var sourceErr *config.SourceReadError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error = %v, want SourceReadError", err)
	}
	if sourceErr.Parent != "/home/user/.tmux.conf" || sourceErr.Directive != "source missing.conf" ||
		sourceErr.Target != "/home/user/missing.conf" {
		t.Fatalf("error context = %#v", sourceErr)
	}
}
