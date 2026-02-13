package manager_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func setupInstalledPlugin(t *testing.T, pluginDir, name string) {
	t.Helper()
	dir := filepath.Join(pluginDir, name)
	os.MkdirAll(dir, 0o755)
}

func TestUpdateAll(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")
	setupInstalledPlugin(t, pluginDir, "tmux-yank")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	puller.Output = "Already up to date."
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, "tmux-sensible")] = true
	validator.Valid[filepath.Join(pluginDir, "tmux-yank")] = true
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		{Name: "tmux-sensible"},
		{Name: "tmux-yank"},
	}

	mgr.Update(context.Background(), plugins, []string{"all"})

	if len(puller.Calls) != 2 {
		t.Errorf("expected 2 pull calls, got %d", len(puller.Calls))
	}

	foundHeader := false
	for _, msg := range output.OkMsgs {
		if msg == "Updating all plugins!" {
			foundHeader = true
		}
	}
	if !foundHeader {
		t.Error("expected 'Updating all plugins!' message")
	}
}

func TestUpdateSpecific(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	puller.Output = "Already up to date."
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, "tmux-sensible")] = true
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	plugins := []plugin.Plugin{
		{Name: "tmux-sensible"},
		{Name: "tmux-yank"},
	}

	mgr.Update(context.Background(), plugins, []string{"tmux-sensible"})

	if len(puller.Calls) != 1 {
		t.Errorf("expected 1 pull call, got %d", len(puller.Calls))
	}
}

func TestUpdateNotInstalled(t *testing.T) {
	pluginDir := setupTestDir(t)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	mgr.Update(context.Background(), nil, []string{"tmux-foo"})

	found := false
	for _, msg := range output.ErrMsgs {
		if msg == "tmux-foo not installed!" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'not installed' error, got: %v", output.ErrMsgs)
	}
}

func TestUpdateOutputIndented(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	puller.Output = "Updating abc123..def456\nFast-forward"
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, "tmux-sensible")] = true
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	mgr.Update(context.Background(), []plugin.Plugin{{Name: "tmux-sensible"}}, []string{"all"})

	foundIndented := false
	for _, msg := range output.OkMsgs {
		if len(msg) > 4 && msg[:6] == "    | " {
			foundIndented = true
		}
	}
	if !foundIndented {
		t.Errorf("expected indented output, got: %v", output.OkMsgs)
	}
}

func TestUpdatePullFails(t *testing.T) {
	pluginDir := setupTestDir(t)
	setupInstalledPlugin(t, pluginDir, "tmux-sensible")

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	puller.Err = errors.New("pull failed")
	puller.Output = "error: something went wrong"
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, "tmux-sensible")] = true
	output := ui.NewMockOutput()

	mgr := manager.New(pluginDir, cloner, puller, validator, output)

	mgr.Update(context.Background(), []plugin.Plugin{{Name: "tmux-sensible"}}, []string{"all"})

	found := false
	for _, msg := range output.ErrMsgs {
		if msg == "  \"tmux-sensible\" update fail" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected update fail message, got: %v", output.ErrMsgs)
	}
}
