package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/ui"
)

func TestSourceExecutesTmuxFiles(t *testing.T) {
	pluginDir := setupTestDir(t)
	pDir := filepath.Join(pluginDir, "tmux-test")
	os.MkdirAll(pDir, 0o755)

	marker := filepath.Join(t.TempDir(), "sourced")
	script := filepath.Join(pDir, "test.tmux")
	os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755)

	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
	failures := mgr.Source(context.Background(), []plug.Plugin{{Name: "tmux-test", DirName: "tmux-test"}})

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected *.tmux file to be executed")
	}
	if len(failures) != 0 {
		t.Errorf("expected no failures, got %v", failures)
	}
}

func TestSourceUsesDirName(t *testing.T) {
	pluginDir := setupTestDir(t)
	p := mustParsePlugin(t, "catppuccin/tmux")
	pDir := filepath.Join(pluginDir, p.DirName)
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(t.TempDir(), "sourced")
	script := filepath.Join(pDir, "test.tmux")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
	failures := mgr.Source(context.Background(), []plug.Plugin{p})

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected DirName/*.tmux file to be executed")
	}
	if len(failures) != 0 {
		t.Errorf("expected no failures, got %v", failures)
	}
}

func TestSourceFailurePersistsNameAndDirName(t *testing.T) {
	pluginDir := setupTestDir(t)
	p := mustParsePlugin(t, "catppuccin/tmux")
	pDir := filepath.Join(pluginDir, p.DirName)
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pDir, "fail.tmux"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
	failures := mgr.Source(context.Background(), []plug.Plugin{p})

	if len(failures) != 1 {
		t.Fatalf("failures = %v", failures)
	}
	if failures[0].Name != "tmux" || failures[0].DirName != "tmux-87a1216f1f68" {
		t.Errorf("failure identity = %#v", failures[0])
	}
}

func TestSourceReportsPluginErrorOutput(t *testing.T) {
	pluginDir := setupTestDir(t)
	pDir := filepath.Join(pluginDir, "tmux-test")
	os.MkdirAll(pDir, 0o755)

	script := filepath.Join(pDir, "test.tmux")
	os.WriteFile(script, []byte("#!/bin/sh\necho 'boom: missing dependency' >&2\nexit 1\n"), 0o755)

	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
	failures := mgr.Source(context.Background(), []plug.Plugin{{Name: "tmux-test", DirName: "tmux-test"}})

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if failures[0].Name != "tmux-test" {
		t.Errorf("expected failure name 'tmux-test', got %q", failures[0].Name)
	}
	if !strings.Contains(failures[0].Message, "test.tmux") {
		t.Errorf("expected message to name the file, got %q", failures[0].Message)
	}
	if !strings.Contains(failures[0].Message, "boom: missing dependency") {
		t.Errorf("expected plugin stderr in message, got %q", failures[0].Message)
	}
}

func TestSourceFallsBackToShebangInterpreter(t *testing.T) {
	cases := []struct {
		name    string
		shebang string
	}{
		{"env_bash", "#!/nonexistent/env bash"},
		{"direct_sh", "#!/nonexistent/bin/sh"},
		{"env_sh", "#!/nonexistent/env sh"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pluginDir := setupTestDir(t)
			pDir := filepath.Join(pluginDir, "tmux-test")
			os.MkdirAll(pDir, 0o755)

			marker := filepath.Join(t.TempDir(), "sourced")
			script := filepath.Join(pDir, "test.tmux")
			os.WriteFile(script, []byte(tc.shebang+"\ntouch "+marker+"\n"), 0o755)

			mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
			failures := mgr.Source(context.Background(), []plug.Plugin{{Name: "tmux-test", DirName: "tmux-test"}})

			if _, err := os.Stat(marker); err != nil {
				t.Error("expected shebang fallback to execute script")
			}
			if len(failures) != 0 {
				t.Errorf("expected no failures, got %v", failures)
			}
		})
	}
}

func TestSourceFallsBackToShellForShebanglessScript(t *testing.T) {
	pluginDir := setupTestDir(t)
	pDir := filepath.Join(pluginDir, "tmux-test")
	os.MkdirAll(pDir, 0o755)

	// Script with NO shebang line (starts directly with shell code).
	// Direct exec fails with ENOEXEC; tpack should fall back to running
	// it through a shell, matching TPM's run-shell behavior.
	marker := filepath.Join(t.TempDir(), "sourced")
	script := filepath.Join(pDir, "test.tmux")
	os.WriteFile(script, []byte("touch "+marker+"\n"), 0o755)

	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())
	failures := mgr.Source(context.Background(), []plug.Plugin{{Name: "tmux-test", DirName: "tmux-test"}})

	if _, err := os.Stat(marker); err != nil {
		t.Error("expected shebang-less script to be executed via shell fallback")
	}
	if len(failures) != 0 {
		t.Errorf("expected no failures, got %v", failures)
	}
}

func TestSourceSkipsNonExistentDir(t *testing.T) {
	pluginDir := setupTestDir(t)
	mgr := manager.New(mustRoot(t, pluginDir), git.NewMockCloner(), git.NewMockPuller(), git.NewMockValidator(), ui.NewMockOutput())

	failures := mgr.Source(context.Background(), []plug.Plugin{{Name: "nonexistent", DirName: "nonexistent"}})
	if len(failures) != 0 {
		t.Errorf("expected no failures for missing dir, got %v", failures)
	}
}
