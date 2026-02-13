package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBinary(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		result := findBinary()
		if result == "" {
			t.Fatal("findBinary() returned empty string")
		}
	})

	t.Run("returned path exists on disk", func(t *testing.T) {
		result := findBinary()
		if _, err := os.Stat(result); err != nil {
			t.Errorf("findBinary() returned %q which does not exist: %v", result, err)
		}
	})
}

func TestFindBinary_WithCurrentDirEnv(t *testing.T) {
	// In test context os.Executable succeeds, so findBinary always returns
	// the test binary path regardless of CURRENT_DIR. We verify that
	// findBinary still returns a valid, existing path when CURRENT_DIR is set.
	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "tpm-go")
	if err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CURRENT_DIR", tmpDir)

	result := findBinary()
	if result == "" {
		t.Fatal("findBinary() returned empty string with CURRENT_DIR set")
	}
	if _, err := os.Stat(result); err != nil {
		t.Errorf("findBinary() returned %q which does not exist: %v", result, err)
	}
}
