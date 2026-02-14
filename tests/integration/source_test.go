package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/plug"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func TestSourceExecutesTmuxFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping source execution test in -short mode")
	}

	pluginDir, _ := setupIntegrationDir(t)
	markerFile := filepath.Join(t.TempDir(), "source-marker")

	// Create plugin directory and an executable .tmux file inside it.
	pluginSubDir := filepath.Join(pluginDir, "test-plugin")
	if err := os.MkdirAll(pluginSubDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tmuxFile := filepath.Join(pluginSubDir, "test-plugin.tmux")
	script := "#!/bin/sh\ntouch " + markerFile + "\n"
	if err := os.WriteFile(tmuxFile, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "test-plugin", Name: "test-plugin", Spec: "test-plugin"},
	}
	mgr.Source(plugins)

	if _, err := os.Stat(markerFile); err != nil {
		t.Errorf("expected marker file to exist after Source(), got: %v", err)
	}
}

func TestSourceSkipsNonExistentPluginDir(t *testing.T) {
	pluginDir, _ := setupIntegrationDir(t)

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "nonexistent-plugin", Name: "nonexistent-plugin", Spec: "nonexistent-plugin"},
	}

	// Should not panic or error.
	mgr.Source(plugins)

	if output.HasFailed() {
		t.Errorf("expected no errors for non-existent plugin dir, got: %v", output.ErrMsgs)
	}
}
