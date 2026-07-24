package manager_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Name: "tmux-sensible", DirName: "tmux-sensible"},
		{Name: "tmux-yank", DirName: "tmux-yank"},
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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Name: "tmux-sensible", DirName: "tmux-sensible"},
		{Name: "tmux-yank", DirName: "tmux-yank"},
	}

	mgr.Update(context.Background(), plugins, []string{"tmux-sensible"})

	if len(puller.Calls) != 1 {
		t.Errorf("expected 1 pull call, got %d", len(puller.Calls))
	}
}

func TestUpdateSpecificSelectsExactRepositoryNameAndUsesDirName(t *testing.T) {
	pluginDir := setupTestDir(t)
	catppuccin := mustParsePlugin(t, "catppuccin/tmux")
	dracula := mustParsePlugin(t, "dracula/tmux")
	setupInstalledPlugin(t, pluginDir, dracula.DirName)

	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, dracula.DirName)] = true
	output := ui.NewMockOutput()
	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), puller, validator, output)

	mgr.Update(context.Background(), []plug.Plugin{catppuccin, dracula}, []string{"dracula/tmux"})

	if len(puller.Calls) != 1 {
		t.Fatalf("pull calls = %d, errors = %v", len(puller.Calls), output.ErrMsgs)
	}
	wantDir := filepath.Join(pluginDir, "tmux-e74ab6318c07")
	if puller.Calls[0].Dir != wantDir {
		t.Errorf("pull directory = %q, want %q", puller.Calls[0].Dir, wantDir)
	}
	foundName := false
	for _, msg := range output.OkMsgs {
		if msg == "  \"dracula/tmux\" update success" {
			foundName = true
		}
	}
	if !foundName {
		t.Errorf("update output does not retain Name: %v", output.OkMsgs)
	}
}

func TestUpdateSpecificRejectsNameAbsentFromConfig(t *testing.T) {
	pluginDir := setupTestDir(t)
	p := mustParsePlugin(t, "catppuccin/tmux")
	setupInstalledPlugin(t, pluginDir, p.DirName)

	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	validator.Valid[filepath.Join(pluginDir, p.DirName)] = true
	output := ui.NewMockOutput()
	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), puller, validator, output)

	mgr.Update(context.Background(), []plug.Plugin{p}, []string{"tmux"})

	if len(puller.Calls) != 0 {
		t.Fatalf("unconfigured name caused %d pull calls", len(puller.Calls))
	}
	if len(output.ErrMsgs) != 1 || output.ErrMsgs[0] != "tmux not configured!" {
		t.Fatalf("errors = %v, want requested exact name", output.ErrMsgs)
	}
}

func TestUpdateNotInstalled(t *testing.T) {
	pluginDir := setupTestDir(t)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	mgr.Update(context.Background(), []plug.Plugin{{Name: "tmux-foo", DirName: "tmux-foo"}}, []string{"tmux-foo"})

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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	mgr.Update(context.Background(), []plug.Plugin{{Name: "tmux-sensible", DirName: "tmux-sensible"}}, []string{"all"})

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

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	mgr.Update(context.Background(), []plug.Plugin{{Name: "tmux-sensible", DirName: "tmux-sensible"}}, []string{"all"})

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
