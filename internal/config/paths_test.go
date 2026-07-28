package config_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func testEnv() config.Env {
	return config.Env{Home: "/home/user", XDGConfigHome: "/home/user/.config"}
}

func TestResolvePathsEmptyHomeFails(t *testing.T) {
	_, err := config.ResolvePaths(tmux.NewMockRunner(), config.NewMockFS(), config.Env{})
	if err == nil {
		t.Fatal("expected error for empty home, got nil")
	}
}

func TestResolvePathsUsesExplicitConfigOverride(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Options[config.ConfigPathOption] = "/custom/tmux.conf"
	runner.Formats["#{config_files}"] = "/ignored/tmux.conf"
	fs := config.NewMockFS()
	fs.Files["/custom/tmux.conf"] = ""
	fs.Files["/ignored/tmux.conf"] = ""

	paths, err := config.ResolvePaths(runner, fs, testEnv())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/custom/tmux.conf"}
	if paths.TmuxConf != "/custom/tmux.conf" || !reflect.DeepEqual(paths.TmuxConfs, want) {
		t.Fatalf("paths = %#v, want override as the only root and writable target", paths)
	}
}

func TestResolvePathsRejectsInvalidExplicitConfigOverride(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		nonRegular bool
	}{
		{name: "relative", value: "relative/tmux.conf"},
		{name: "missing", value: "/missing/tmux.conf"},
		{name: "non-regular", value: "/custom/tmux.conf", nonRegular: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tmux.NewMockRunner()
			runner.Options[config.ConfigPathOption] = tt.value
			fs := config.NewMockFS()
			if tt.nonRegular {
				fs.Files[tt.value] = ""
				fs.NonRegular[tt.value] = true
			}

			_, err := config.ResolvePaths(runner, fs, testEnv())
			if err == nil {
				t.Fatal("ResolvePaths returned no error")
			}
			if !strings.Contains(err.Error(), config.ConfigPathOption) || !strings.Contains(err.Error(), tt.value) {
				t.Fatalf("error = %q, want option name and rejected value", err)
			}
		})
	}
}

func TestResolvePathsUsesActiveTmuxConfigFiles(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{config_files}"] = "/etc/tmux.conf, /custom/one.conf,/custom/./one.conf,/custom/two.conf"
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = ""
	fs.Files["/custom/one.conf"] = ""
	fs.Files["/custom/two.conf"] = ""

	paths, err := config.ResolvePaths(runner, fs, testEnv())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/etc/tmux.conf", "/custom/one.conf", "/custom/two.conf"}
	if !reflect.DeepEqual(paths.TmuxConfs, want) || paths.TmuxConf != "/custom/two.conf" {
		t.Fatalf("paths = %#v, want roots %v and writable last custom root", paths, want)
	}
}

func TestResolvePathsRejectsInvalidActiveCustomConfig(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		nonRegular bool
	}{
		{name: "relative", value: "relative/tmux.conf"},
		{name: "missing", value: "/missing/custom.conf"},
		{name: "non-regular", value: "/custom/directory", nonRegular: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tmux.NewMockRunner()
			runner.Formats["#{config_files}"] = tt.value
			fs := config.NewMockFS()
			fs.Files["/home/user/.tmux.conf"] = "fallback must not be used"
			if tt.nonRegular {
				fs.Files[tt.value] = ""
				fs.NonRegular[tt.value] = true
			}

			_, err := config.ResolvePaths(runner, fs, testEnv())
			if err == nil {
				t.Fatal("ResolvePaths returned no error")
			}
			if !strings.Contains(err.Error(), "#{config_files}") || !strings.Contains(err.Error(), tt.value) {
				t.Fatalf("error = %q, want config_files and rejected value", err)
			}
		})
	}
}

func TestResolvePathsAllowsMissingActiveDefaultCandidates(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{config_files}"] = "/etc/tmux.conf,/home/user/.tmux.conf,/home/user/.config/tmux/tmux.conf,/custom/active.conf"
	fs := config.NewMockFS()
	fs.Files["/custom/active.conf"] = ""

	paths, err := config.ResolvePaths(runner, fs, testEnv())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/custom/active.conf"}
	if !reflect.DeepEqual(paths.TmuxConfs, want) || paths.TmuxConf != "/custom/active.conf" {
		t.Fatalf("paths = %#v, want only existing custom root", paths)
	}
}

func TestResolvePathsAllowsMissingPackagedSystemConfig(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{config_files}"] = "/home/linuxbrew/.linuxbrew/etc/tmux.conf,/home/user/.config/tmux/tmux.conf"
	fs := config.NewMockFS()
	fs.Files["/home/user/.config/tmux/tmux.conf"] = ""

	paths, err := config.ResolvePaths(runner, fs, testEnv())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/home/user/.config/tmux/tmux.conf"}
	if !reflect.DeepEqual(paths.TmuxConfs, want) || paths.TmuxConf != "/home/user/.config/tmux/tmux.conf" {
		t.Fatalf("paths = %#v, want only existing XDG root", paths)
	}
}

func TestResolvePathsActiveSystemConfigOnlyReturnsErrNoTmuxConf(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Formats["#{config_files}"] = "/etc/tmux.conf"
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = ""
	fs.Files["/home/user/.tmux.conf"] = "inactive fallback"

	_, err := config.ResolvePaths(runner, fs, testEnv())
	var noConf *config.ErrNoTmuxConf
	if !errors.As(err, &noConf) {
		t.Fatalf("error = %v, want ErrNoTmuxConf", err)
	}
	want := []string{"/etc/tmux.conf"}
	if !reflect.DeepEqual(noConf.Searched, want) {
		t.Fatalf("Searched = %v, want active roots %v", noConf.Searched, want)
	}
}

func TestResolvePathsKeepsAllExistingDefaultsWithCustomXDG(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Errors["ExpandFormat:#{config_files}"] = errors.New("no server")
	fs := config.NewMockFS()
	fs.Files["/etc/tmux.conf"] = "set -g @plugin owner/system"
	fs.Files["/home/user/.tmux.conf"] = "set -g @plugin owner/legacy"
	fs.Files["/custom/xdg/tmux/tmux.conf"] = "set -g @plugin owner/custom"
	fs.Files["/home/user/.config/tmux/tmux.conf"] = "set -g @plugin owner/fallback"

	paths, err := config.ResolvePaths(runner, fs, config.Env{Home: "/home/user", XDGConfigHome: "/custom/xdg"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/etc/tmux.conf",
		"/home/user/.tmux.conf",
		"/custom/xdg/tmux/tmux.conf",
		"/home/user/.config/tmux/tmux.conf",
	}
	if !reflect.DeepEqual(paths.TmuxConfs, want) {
		t.Fatalf("TmuxConfs = %v, want %v", paths.TmuxConfs, want)
	}
	if paths.TmuxConf != "/custom/xdg/tmux/tmux.conf" {
		t.Fatalf("TmuxConf = %q, want current-preference XDG target", paths.TmuxConf)
	}
}

func TestResolvePathsRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		env     config.Env
		option  string
		envPath string
	}{
		{name: "relative home", env: config.Env{Home: "relative"}},
		{name: "relative XDG config", env: config.Env{Home: "/home/user", XDGConfigHome: "relative"}},
		{name: "relative XDG data", env: config.Env{Home: "/home/user", XDGDataHome: "relative"}},
		{name: "relative XDG state", env: config.Env{Home: "/home/user", XDGStateHome: "relative"}},
		{name: "relative option", env: testEnv(), option: "."},
		{name: "normalised root option", env: testEnv(), option: "/."},
		{name: "relative tmux environment", env: testEnv(), envPath: "../plugins"},
		{name: "filesystem root tmux environment", env: testEnv(), envPath: "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			if tt.option != "" {
				m.Options[config.PluginPathOption] = tt.option
			}
			if tt.envPath != "" {
				m.Environment[config.PluginPathEnvVar] = tt.envPath
			}
			fs := config.NewMockFS()
			fs.Files["/home/user/.tmux.conf"] = ""
			if _, err := config.ResolvePaths(m, fs, tt.env); err == nil {
				t.Fatal("ResolvePaths returned no error")
			}
		})
	}
}

func TestResolvePathsRejectsPresentEmptyPluginPaths(t *testing.T) {
	tests := []struct {
		name        string
		optionSet   bool
		option      string
		environment map[string]string
	}{
		{
			name:        "empty option",
			optionSet:   true,
			environment: map[string]string{config.PluginPathEnvVar: "/lower-precedence"},
		},
		{
			name:        "whitespace option",
			optionSet:   true,
			option:      " \t ",
			environment: map[string]string{config.PluginPathEnvVar: "/lower-precedence"},
		},
		{
			name:        "empty tpack environment",
			environment: map[string]string{config.PluginPathEnvVar: "", config.LegacyPluginPathEnvVar: "/lower-precedence"},
		},
		{
			name:        "whitespace tpack environment",
			environment: map[string]string{config.PluginPathEnvVar: " \n ", config.LegacyPluginPathEnvVar: "/lower-precedence"},
		},
		{
			name:        "empty legacy environment",
			environment: map[string]string{config.LegacyPluginPathEnvVar: ""},
		},
		{
			name:        "whitespace legacy environment",
			environment: map[string]string{config.LegacyPluginPathEnvVar: "  "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tmux.NewMockRunner()
			if tt.optionSet {
				runner.Options[config.PluginPathOption] = tt.option
			}
			for name, value := range tt.environment {
				runner.Environment[name] = value
			}
			fs := config.NewMockFS()
			fs.Files["/home/user/.tmux.conf"] = ""

			_, err := config.ResolvePaths(runner, fs, testEnv())
			var rootErr *plug.RootError
			if !errors.As(err, &rootErr) {
				t.Fatalf("ResolvePaths error = %v, want *plug.RootError", err)
			}
		})
	}
}

func TestResolvePathsConfDiscovery(t *testing.T) {
	tests := []struct {
		name  string
		env   config.Env
		files []string
		want  string
	}{
		{
			name:  "XDG conf wins when both exist",
			env:   testEnv(),
			files: []string{"/home/user/.config/tmux/tmux.conf", "/home/user/.tmux.conf"},
			want:  "/home/user/.config/tmux/tmux.conf",
		},
		{
			name: "hardcoded ~/.config tried when XDG_CONFIG_HOME points elsewhere",
			env:  config.Env{Home: "/home/user", XDGConfigHome: "/custom/xdg"},
			files: []string{
				"/home/user/.config/tmux/tmux.conf",
			},
			want: "/home/user/.config/tmux/tmux.conf",
		},
		{
			name:  "falls back to ~/.tmux.conf",
			env:   testEnv(),
			files: []string{"/home/user/.tmux.conf"},
			want:  "/home/user/.tmux.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := config.NewMockFS()
			for _, f := range tt.files {
				fs.Files[f] = ""
			}

			p, err := config.ResolvePaths(tmux.NewMockRunner(), fs, tt.env)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.TmuxConf != tt.want {
				t.Errorf("TmuxConf = %q, want %q", p.TmuxConf, tt.want)
			}
		})
	}
}

func TestResolvePathsNoConfFailsClosed(t *testing.T) {
	_, err := config.ResolvePaths(tmux.NewMockRunner(), config.NewMockFS(), testEnv())

	var noConf *config.ErrNoTmuxConf
	if !errors.As(err, &noConf) {
		t.Fatalf("expected ErrNoTmuxConf, got %v", err)
	}
	// Default XDG config home collapses the two XDG candidates into one entry.
	want := []string{"/etc/tmux.conf", "/home/user/.tmux.conf", "/home/user/.config/tmux/tmux.conf"}
	if len(noConf.Searched) != len(want) {
		t.Fatalf("Searched = %v, want %v", noConf.Searched, want)
	}
	for i := range want {
		if noConf.Searched[i] != want[i] {
			t.Errorf("Searched[%d] = %q, want %q", i, noConf.Searched[i], want[i])
		}
	}
}

func TestResolvePathsNoConfSearchListWithCustomXDG(t *testing.T) {
	env := config.Env{Home: "/home/user", XDGConfigHome: "/custom/xdg"}
	_, err := config.ResolvePaths(tmux.NewMockRunner(), config.NewMockFS(), env)

	var noConf *config.ErrNoTmuxConf
	if !errors.As(err, &noConf) {
		t.Fatalf("expected ErrNoTmuxConf, got %v", err)
	}
	want := []string{
		"/etc/tmux.conf",
		"/home/user/.tmux.conf",
		"/custom/xdg/tmux/tmux.conf",
		"/home/user/.config/tmux/tmux.conf",
	}
	if len(noConf.Searched) != len(want) {
		t.Fatalf("Searched = %v, want %v", noConf.Searched, want)
	}
	for i := range want {
		if noConf.Searched[i] != want[i] {
			t.Errorf("Searched[%d] = %q, want %q", i, noConf.Searched[i], want[i])
		}
	}
}

func TestResolvePathsPluginPathPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		options    map[string]string
		envVars    map[string]string
		files      []string
		wantPath   string
		wantSource config.PathSource
	}{
		{
			name:       "option beats both env vars",
			options:    map[string]string{"@tpack-plugin-path": "/opt/plugins"},
			envVars:    map[string]string{"TPACK_PLUGIN_PATH": "/env/tpack/", "TMUX_PLUGIN_MANAGER_PATH": "/env/legacy/"},
			files:      []string{"/home/user/.tmux.conf"},
			wantPath:   "/opt/plugins/",
			wantSource: config.SourceOption,
		},
		{
			name:       "option expands tilde",
			options:    map[string]string{"@tpack-plugin-path": "~/my/plugins"},
			files:      []string{"/home/user/.tmux.conf"},
			wantPath:   "/home/user/my/plugins/",
			wantSource: config.SourceOption,
		},
		{
			name:       "TPACK_PLUGIN_PATH beats legacy env var",
			envVars:    map[string]string{"TPACK_PLUGIN_PATH": "/env/tpack", "TMUX_PLUGIN_MANAGER_PATH": "/env/legacy/"},
			files:      []string{"/home/user/.tmux.conf"},
			wantPath:   "/env/tpack/",
			wantSource: config.SourceEnvTpack,
		},
		{
			name:       "legacy env var still works",
			envVars:    map[string]string{"TMUX_PLUGIN_MANAGER_PATH": "/env/legacy"},
			files:      []string{"/home/user/.tmux.conf"},
			wantPath:   "/env/legacy/",
			wantSource: config.SourceEnvLegacy,
		},
		{
			name:       "legacy conf with legacy dir detected",
			files:      []string{"/home/user/.tmux.conf", "/home/user/.tmux/plugins"},
			wantPath:   "/home/user/.tmux/plugins/",
			wantSource: config.SourceDetectedLegacy,
		},
		{
			name:       "XDG conf with XDG dir detected",
			files:      []string{"/home/user/.config/tmux/tmux.conf", "/home/user/.config/tmux/plugins"},
			wantPath:   "/home/user/.config/tmux/plugins/",
			wantSource: config.SourceDetectedXDGConfig,
		},
		{
			name: "both dirs exist, XDG conf prefers XDG dir",
			files: []string{
				"/home/user/.config/tmux/tmux.conf",
				"/home/user/.config/tmux/plugins",
				"/home/user/.tmux/plugins",
			},
			wantPath:   "/home/user/.config/tmux/plugins/",
			wantSource: config.SourceDetectedXDGConfig,
		},
		{
			name: "both dirs exist, legacy conf prefers legacy dir",
			files: []string{
				"/home/user/.tmux.conf",
				"/home/user/.config/tmux/plugins",
				"/home/user/.tmux/plugins",
			},
			wantPath:   "/home/user/.tmux/plugins/",
			wantSource: config.SourceDetectedLegacy,
		},
		{
			name: "XDG conf falls back to existing legacy dir",
			files: []string{
				"/home/user/.config/tmux/tmux.conf",
				"/home/user/.tmux/plugins",
			},
			wantPath:   "/home/user/.tmux/plugins/",
			wantSource: config.SourceDetectedLegacy,
		},
		{
			name:       "fresh install gets XDG data default",
			files:      []string{"/home/user/.tmux.conf"},
			wantPath:   "/home/user/.local/share/tmux/plugins/",
			wantSource: config.SourceDefaultXDGData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tmux.NewMockRunner()
			for k, v := range tt.options {
				m.Options[k] = v
			}
			for k, v := range tt.envVars {
				m.Environment[k] = v
			}
			fs := config.NewMockFS()
			for _, f := range tt.files {
				fs.Files[f] = ""
			}

			p, err := config.ResolvePaths(m, fs, testEnv())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.PluginPath.String() != tt.wantPath {
				t.Errorf("PluginPath = %q, want %q", p.PluginPath, tt.wantPath)
			}
			if p.PluginPathSource != tt.wantSource {
				t.Errorf("PluginPathSource = %d, want %d", p.PluginPathSource, tt.wantSource)
			}
		})
	}
}

func TestResolvePathsCustomXDGDataHome(t *testing.T) {
	env := config.Env{Home: "/home/user", XDGConfigHome: "/home/user/.config", XDGDataHome: "/custom/data"}
	fs := config.NewMockFS()
	fs.Files["/home/user/.tmux.conf"] = ""

	p, err := config.ResolvePaths(tmux.NewMockRunner(), fs, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PluginPath.String() != "/custom/data/tmux/plugins/" {
		t.Errorf("PluginPath = %q, want /custom/data/tmux/plugins/", p.PluginPath)
	}
}
