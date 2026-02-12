package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestInstallRealPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		plugin.ParseSpec(tmuxExamplePlugin),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	// Verify the plugin was cloned.
	dir := filepath.Join(pluginDir, "tmux-example-plugin")
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("plugin directory not created: %v", err)
	}

	if output.HasFailed() {
		t.Errorf("install reported failure: %v", output.ErrMsgs)
	}

	// Install again should say "already installed".
	output2 := ui.NewMockOutput()
	mgr2 := manager.New(pluginDir, cloner, puller, validator, output2)
	mgr2.Install(ctx, plugins)

	found := false
	for _, msg := range output2.OkMsgs {
		if msg == "Already installed \"tmux-example-plugin\"" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Already installed' on second install, got: %v", output2.OkMsgs)
	}
}

func TestInstallMultiplePlugins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		plugin.ParseSpec(tmuxExamplePlugin),
		plugin.ParseSpec("tmux-plugins/tmux-sensible"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	for _, name := range []string{"tmux-example-plugin", "tmux-sensible"} {
		dir := filepath.Join(pluginDir, name)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("plugin %s not installed: %v", name, err)
		}
	}
}

func TestInstallNonexistentPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in -short mode")
	}
	skipIfNoGit(t)

	pluginDir, _ := setupIntegrationDir(t)

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		plugin.ParseSpec("nonexistent-user/nonexistent-plugin-xyz-abc-123"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	mgr.Install(ctx, plugins)

	if !output.HasFailed() {
		t.Error("expected failure for nonexistent plugin")
	}
}

func TestConfigGatherPluginsFromFile(t *testing.T) {
	_, confFile := setupIntegrationDir(t)

	writeConf(t, confFile, `
set -g @plugin "tmux-plugins/tpm"
set -g @plugin "tmux-plugins/tmux-sensible"
set -g @plugin "user/repo#develop"
`)

	fs := config.RealFS{}
	plugins, err := config.GatherPlugins(
		&noopRunner{},
		fs,
		confFile,
		os.Getenv("HOME"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(plugins))
	}
	if plugins[2].Branch != "develop" {
		t.Errorf("expected branch 'develop', got %q", plugins[2].Branch)
	}
}

// noopRunner is a tmux.Runner that returns empty values.
type noopRunner struct{}

func (n *noopRunner) ShowOption(string) (string, error)          { return "", nil }
func (n *noopRunner) ShowEnvironment(string) (string, error)     { return "", nil }
func (n *noopRunner) SetEnvironment(string, string) error        { return nil }
func (n *noopRunner) BindKey(string, string) error               { return nil }
func (n *noopRunner) SourceFile(string) error                    { return nil }
func (n *noopRunner) DisplayMessage(string) error                { return nil }
func (n *noopRunner) RunShell(string) error                      { return nil }
func (n *noopRunner) CommandPrompt(string, string) error         { return nil }
func (n *noopRunner) Version() (string, error)                   { return "tmux 3.4", nil }
func (n *noopRunner) StartServer() error                         { return nil }
func (n *noopRunner) ShowWindowOption(string) (string, error)    { return "", nil }
func (n *noopRunner) SetOption(string, string) error             { return nil }
