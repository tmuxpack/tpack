package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

func promptTestConfig(t *testing.T, content string) *config.Config {
	t.Helper()
	base := t.TempDir()
	conf := filepath.Join(base, "tmux.conf")
	if err := os.WriteFile(conf, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	pluginPath := mustRoot(t, filepath.Join(base, "plugins"))
	return &config.Config{
		TmuxConf:   conf,
		PluginPath: pluginPath,
		Paths: config.Paths{
			TmuxConf:      conf,
			PluginPath:    pluginPath,
			Home:          base,
			XDGConfigHome: filepath.Join(base, ".config"),
		},
	}
}

func TestRunUpdatePromptReturnsPaneTransportFailureThroughSharedReporter(t *testing.T) {
	runner := tmux.NewMockRunner()
	runner.Errors["RunShell"] = errors.New("pane unavailable")
	output := ui.NewReporter(ui.NewTmuxSink(runner))

	runUpdatePrompt(runner, promptTestConfig(t, ""), output)

	var transport *ui.TransportError
	if err := outputResult(output); !errors.As(err, &transport) {
		t.Fatalf("outputResult() = %v, want pane transport error", err)
	}
}

func TestRunUpdatePromptStopsWhenRequiredSourceCannotBeRead(t *testing.T) {
	runner := tmux.NewMockRunner()
	output := ui.NewMockOutput()

	runUpdatePrompt(runner, promptTestConfig(t, "source-file missing.conf\n"), output)

	if output.Result() == nil {
		t.Fatal("prompt result succeeded after required source failure")
	}
	if len(output.ErrMsgs) != 1 || !strings.Contains(output.ErrMsgs[0], "cannot read required source") {
		t.Fatalf("errors = %q, want required source error", output.ErrMsgs)
	}
	for _, call := range runner.Calls {
		if call.Method == "CommandPrompt" {
			t.Fatal("CommandPrompt called after plugin gathering failed")
		}
	}
}

func TestRunUpdatePromptMigrationFailureStopsBeforePrompt(t *testing.T) {
	cfg := promptTestConfig(t, `set -g @plugin "catppuccin/tmux"`)
	legacyPath := filepath.Join(cfg.PluginPath.String(), "tmux")
	if err := os.MkdirAll(legacyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	runner := tmux.NewMockRunner()
	output := ui.NewMockOutput()

	runUpdatePrompt(runner, cfg, output)

	if output.Result() == nil {
		t.Fatal("prompt succeeded after migration failure")
	}
	if len(output.ErrMsgs) != 1 || !strings.Contains(output.ErrMsgs[0], "migrate legacy plugins") {
		t.Fatalf("errors = %q, want one migration error", output.ErrMsgs)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy path changed after migration failure: %v", err)
	}
	wantSource := "SourceFile:" + cfg.TmuxConf
	sourced := false
	for _, call := range runner.Calls {
		sourced = sourced || call.Method+":"+strings.Join(call.Args, ":") == wantSource
		if call.Method == "CommandPrompt" {
			t.Fatal("CommandPrompt called after migration failure")
		}
	}
	if !sourced {
		t.Fatal("tmux configuration was not sourced before migration")
	}
}

func TestListInstalledPluginsUsesDirName(t *testing.T) {
	cfg := promptTestConfig(t, "set -g @plugin \"catppuccin/tmux\"\n")
	pluginDir := filepath.Join(cfg.PluginPath.String(), "tmux-87a1216f1f68")
	runGitCommand(t, "", "init", pluginDir)
	output := ui.NewMockOutput()

	if ok := listInstalledPlugins(tmux.NewMockRunner(), cfg, output); !ok {
		t.Fatalf("listInstalledPlugins failed: %v", output.ErrMsgs)
	}
	for _, msg := range output.OkMsgs {
		if msg == "  catppuccin/tmux" {
			return
		}
	}
	t.Fatalf("installed canonical plugin omitted from output: %v", output.OkMsgs)
}

func TestRunUpdatePromptReportsDirectTmuxErrors(t *testing.T) {
	tests := []struct {
		name       string
		errorKey   func(*config.Config) string
		want       string
		wantPrompt bool
	}{
		{
			name:     "source file",
			errorKey: func(cfg *config.Config) string { return "SourceFile:" + cfg.TmuxConf },
			want:     "source tmux config",
		},
		{
			name:       "command prompt",
			errorKey:   func(*config.Config) string { return "CommandPrompt" },
			want:       "open update prompt",
			wantPrompt: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := promptTestConfig(t, "")
			runner := tmux.NewMockRunner()
			runner.Errors[tt.errorKey(cfg)] = errors.New("tmux unavailable")
			output := ui.NewMockOutput()

			runUpdatePrompt(runner, cfg, output)

			if output.Result() == nil {
				t.Fatal("prompt succeeded after direct tmux error")
			}
			if len(output.ErrMsgs) != 1 || !strings.Contains(output.ErrMsgs[0], tt.want) {
				t.Fatalf("errors = %q, want message containing %q", output.ErrMsgs, tt.want)
			}
			promptCalled := false
			for _, call := range runner.Calls {
				promptCalled = promptCalled || call.Method == "CommandPrompt"
			}
			if promptCalled != tt.wantPrompt {
				t.Fatalf("CommandPrompt called = %v, want %v", promptCalled, tt.wantPrompt)
			}
		})
	}
}
