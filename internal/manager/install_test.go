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

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins") + "/"
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return pluginDir
}

func TestInstallNewPlugin(t *testing.T) {
	pluginDir := setupTestDir(t)
	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "tmux-plugins/tmux-sensible", Name: "tmux-sensible", DirName: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	mgr.Install(context.Background(), plugins)

	if len(cloner.Calls) == 0 {
		t.Fatal("expected clone to be called")
	}
	if cloner.Calls[0].URL != "tmux-plugins/tmux-sensible" {
		t.Errorf("clone URL = %q, want raw URL first", cloner.Calls[0].URL)
	}

	found := false
	for _, msg := range output.OkMsgs {
		if msg == "Installing \"tmux-sensible\"" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Installing message, got: %v", output.OkMsgs)
	}
}

func TestInstallSameBasenameRepositoriesUsesDistinctDirectories(t *testing.T) {
	root := mustRoot(t, setupTestDir(t))
	catppuccin := mustParsePlugin(t, "catppuccin/tmux")
	dracula := mustParsePlugin(t, "dracula/tmux")
	cloner := git.NewMockCloner()
	mgr := manager.New(root, cloner, git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())

	mgr.Install(context.Background(), []plug.Plugin{catppuccin, dracula})

	if len(cloner.Calls) != 2 {
		t.Fatalf("clone calls = %d", len(cloner.Calls))
	}
	wantDirs := map[string]bool{
		filepath.Join(root.String(), "tmux-87a1216f1f68"): true,
		filepath.Join(root.String(), "tmux-e74ab6318c07"): true,
	}
	for _, call := range cloner.Calls {
		if !wantDirs[call.Dir] {
			t.Errorf("unexpected clone destination %q", call.Dir)
		}
	}
	if cloner.Calls[0].Dir == cloner.Calls[1].Dir {
		t.Fatalf("clone destinations collide at %q", cloner.Calls[0].Dir)
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	pluginDir := setupTestDir(t)
	// Create plugin directory to simulate already installed.
	pluginPath := filepath.Join(pluginDir, "tmux-sensible")
	os.MkdirAll(pluginPath, 0o755)

	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	validator.Valid[pluginPath] = true
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "tmux-plugins/tmux-sensible", Name: "tmux-sensible", DirName: "tmux-sensible", Spec: "tmux-plugins/tmux-sensible"},
	}

	mgr.Install(context.Background(), plugins)

	if len(cloner.Calls) != 0 {
		t.Error("clone should not be called for already installed plugin")
	}

	found := false
	for _, msg := range output.OkMsgs {
		if msg == "Already installed \"tmux-sensible\"" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Already installed message, got: %v", output.OkMsgs)
	}
}

func TestInstallCloneFailsWithFallback(t *testing.T) {
	pluginDir := setupTestDir(t)
	callCount := 0
	// Use a custom cloner that fails first, succeeds on fallback.
	cloner := &countingCloner{
		failUntil: 1,
		count:     &callCount,
	}
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "user/plugin", Name: "plugin", DirName: "plugin", Spec: "user/plugin"},
	}

	mgr.Install(context.Background(), plugins)

	if callCount != 2 {
		t.Errorf("expected 2 clone calls (raw + fallback), got %d", callCount)
	}

	// Should succeed (second call)
	foundSuccess := false
	for _, msg := range output.OkMsgs {
		if msg == "  \"plugin\" download success" {
			foundSuccess = true
		}
	}
	if !foundSuccess {
		t.Errorf("expected download success, got ok: %v, err: %v", output.OkMsgs, output.ErrMsgs)
	}
}

func TestInstallBothClonesFail(t *testing.T) {
	pluginDir := setupTestDir(t)
	cloner := git.NewMockCloner()
	cloner.Err = errors.New("clone failed")
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "user/plugin", Name: "plugin", DirName: "plugin", Spec: "user/plugin"},
	}

	mgr.Install(context.Background(), plugins)

	found := false
	for _, msg := range output.ErrMsgs {
		if msg == "  \"plugin\" download fail" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected download fail, got err: %v, ok: %v", output.ErrMsgs, output.OkMsgs)
	}
}

func TestInstallWithBranch(t *testing.T) {
	pluginDir := setupTestDir(t)
	cloner := git.NewMockCloner()
	puller := git.NewMockPuller()
	validator := git.NewMockValidator()
	output := ui.NewMockOutput()

	mgr := manager.New(mustRoot(t, pluginDir), cloner, puller, validator, output)

	plugins := []plug.Plugin{
		{Raw: "user/repo", Name: "repo", DirName: "repo", Spec: "user/repo", Branch: "develop"},
	}

	mgr.Install(context.Background(), plugins)

	if len(cloner.Calls) == 0 {
		t.Fatal("expected clone call")
	}
	if cloner.Calls[0].Branch != "develop" {
		t.Errorf("branch = %q, want %q", cloner.Calls[0].Branch, "develop")
	}
}

// countingCloner fails for the first N calls then succeeds.
type countingCloner struct {
	failUntil int
	count     *int
}

func mustParsePlugin(t *testing.T, raw string) plug.Plugin {
	t.Helper()
	p, err := plug.ParseSpec(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (c *countingCloner) Clone(_ context.Context, _ git.CloneOptions) error {
	*c.count++
	if *c.count <= c.failUntil {
		return errors.New("clone failed")
	}
	return nil
}
