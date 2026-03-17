package main

import (
	"bytes"
	"strings"
	"testing"
)

func executeCompletion(t *testing.T, shell string) string {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"completion", shell})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion %s failed: %v", shell, err)
	}
	return buf.String()
}

func TestCompletionBash(t *testing.T) {
	out := executeCompletion(t, "bash")
	if !strings.Contains(out, "bash completion") {
		t.Error("expected bash completion output to contain 'bash completion'")
	}
}

func TestCompletionZsh(t *testing.T) {
	out := executeCompletion(t, "zsh")
	if !strings.Contains(out, "zsh completion") {
		t.Error("expected zsh completion output to contain 'zsh completion'")
	}
}

func TestCompletionFish(t *testing.T) {
	out := executeCompletion(t, "fish")
	if !strings.Contains(out, "fish") {
		t.Error("expected fish completion output to contain 'fish'")
	}
}
