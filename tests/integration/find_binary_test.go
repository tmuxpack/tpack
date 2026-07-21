package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func integrationProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func envWithPath(path string) []string {
	var env []string
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "PATH=") &&
			!strings.HasPrefix(item, "TPACK_AUTO_DOWNLOAD=") &&
			!strings.HasPrefix(item, "TPM_AUTO_DOWNLOAD=") {
			env = append(env, item)
		}
	}
	return append(env, "PATH="+path)
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runFindBinary(t *testing.T, root, pathEnv, downloadBody string, extraEnv ...string) (string, int) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}
	scriptPath := filepath.Join(integrationProjectRoot(t), "lib", "find_binary.sh")
	script := fmt.Sprintf("source %q\n", scriptPath)
	if downloadBody != "" {
		script += fmt.Sprintf("_download_tpack() { %s; }\n", downloadBody)
	}
	script += fmt.Sprintf("_find_tpack %q", root)
	cmd := exec.CommandContext(context.Background(), bash, "-c", script)
	cmd.Env = append(envWithPath(pathEnv), extraEnv...)
	out, runErr := cmd.Output()
	if runErr == nil {
		return strings.TrimSpace(string(out)), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatal(runErr)
	}
	return strings.TrimSpace(string(out)), exitErr.ExitCode()
}

func TestFindBinary(t *testing.T) {
	dirname, err := exec.LookPath("dirname")
	if err != nil {
		t.Skip("dirname not found")
	}
	toolDir := t.TempDir()
	if err := os.Symlink(dirname, filepath.Join(toolDir, "dirname")); err != nil {
		t.Skipf("cannot create dirname symlink: %v", err)
	}

	t.Run("dist binary", func(t *testing.T) {
		root := t.TempDir()
		want := filepath.Join(root, "dist", "tpack")
		writeExecutable(t, want)
		got, status := runFindBinary(t, root, toolDir, "return 1")
		if got != want || status != 0 {
			t.Fatalf("got (%q, %d), want (%q, 0)", got, status, want)
		}
	})

	t.Run("root binary with backslash path", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), `root\name`)
		want := filepath.Join(root, "tpack")
		writeExecutable(t, want)
		got, status := runFindBinary(t, root, toolDir, "return 1")
		if got != want || status != 0 {
			t.Fatalf("got (%q, %d), want (%q, 0)", got, status, want)
		}
	})

	t.Run("PATH binary", func(t *testing.T) {
		pathDir := t.TempDir()
		if err := os.Symlink(dirname, filepath.Join(pathDir, "dirname")); err != nil {
			t.Skipf("cannot create dirname symlink: %v", err)
		}
		want := filepath.Join(pathDir, "tpack")
		writeExecutable(t, want)
		got, status := runFindBinary(t, t.TempDir(), pathDir, "return 1")
		if got != want || status != 0 {
			t.Fatalf("got (%q, %d), want (%q, 0)", got, status, want)
		}
	})

	t.Run("successful download", func(t *testing.T) {
		root := t.TempDir()
		want := filepath.Join(root, "tpack")
		got, status := runFindBinary(t, root, toolDir, "return 0")
		if got != want || status != 0 {
			t.Fatalf("got (%q, %d), want (%q, 0)", got, status, want)
		}
	})

	t.Run("failed download", func(t *testing.T) {
		got, status := runFindBinary(t, t.TempDir(), toolDir, "return 1")
		if got != "" || status == 0 {
			t.Fatalf("got (%q, %d), want empty output and nonzero status", got, status)
		}
	})

	t.Run("disabled download", func(t *testing.T) {
		got, status := runFindBinary(t, t.TempDir(), toolDir, "", "TPACK_AUTO_DOWNLOAD=0")
		if got != "" || status == 0 {
			t.Fatalf("got (%q, %d), want empty output and nonzero status", got, status)
		}
	})
}
