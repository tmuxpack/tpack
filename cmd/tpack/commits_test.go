package main

import "testing"

func TestFlagValue(t *testing.T) {
	args := []string{"--dir", "/tmp/plugin", "--from", "abc123", "--to", "def456", "--name", "test-plugin"}

	tests := []struct {
		flag string
		want string
	}{
		{"--dir", "/tmp/plugin"},
		{"--from", "abc123"},
		{"--to", "def456"},
		{"--name", "test-plugin"},
		{"--missing", ""},
	}

	for _, tt := range tests {
		got := flagValue(args, tt.flag)
		if got != tt.want {
			t.Errorf("flagValue(%q) = %q, want %q", tt.flag, got, tt.want)
		}
	}
}

func TestFlagValue_EqualsFormat(t *testing.T) {
	args := []string{"--dir=/tmp/plugin", "--from=abc123"}

	if got := flagValue(args, "--dir"); got != "/tmp/plugin" {
		t.Errorf("flagValue(--dir) = %q, want %q", got, "/tmp/plugin")
	}
	if got := flagValue(args, "--from"); got != "abc123" {
		t.Errorf("flagValue(--from) = %q, want %q", got, "abc123")
	}
}

func TestFlagValue_LastArgNoValue(t *testing.T) {
	args := []string{"--dir"}

	if got := flagValue(args, "--dir"); got != "" {
		t.Errorf("flagValue(--dir) with no value = %q, want empty", got)
	}
}
