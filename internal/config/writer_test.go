package config

import (
	"os"
	"strings"
	"testing"
)

func TestAppendPlugin(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := "set -g @plugin 'tmux-plugins/tpm'\n"
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AppendPlugin(tmp, "catppuccin/tmux"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	content := string(data)

	if !strings.Contains(content, `set -g @plugin "catppuccin/tmux"`) {
		t.Errorf("expected plugin line in file, got:\n%s", content)
	}

	if !strings.Contains(content, "tmux-plugins/tpm") {
		t.Error("original content was lost")
	}
}

func TestAppendPlugin_NoDuplicate(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g @plugin "catppuccin/tmux"` + "\n"
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AppendPlugin(tmp, "catppuccin/tmux"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	if strings.Count(string(data), "catppuccin/tmux") != 1 {
		t.Error("plugin was added twice")
	}
}

func TestRemovePlugin(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g @plugin "tmux-plugins/tpm"
set -g @plugin "catppuccin/tmux"
set -g @plugin "tmux-plugins/tmux-yank"
`
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemovePlugin(tmp, "catppuccin/tmux"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	content := string(data)

	if strings.Contains(content, "catppuccin/tmux") {
		t.Errorf("expected plugin to be removed, got:\n%s", content)
	}
	if !strings.Contains(content, "tmux-plugins/tpm") {
		t.Error("other plugins should be preserved")
	}
	if !strings.Contains(content, "tmux-plugins/tmux-yank") {
		t.Error("other plugins should be preserved")
	}
}

func TestRemovePlugin_NotFound(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g @plugin "tmux-plugins/tpm"` + "\n"
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemovePlugin(tmp, "nonexistent/plugin"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	if string(data) != initial {
		t.Errorf("file should be unchanged, got:\n%s", string(data))
	}
}

func TestRemovePlugin_SingleQuoted(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g @plugin 'catppuccin/tmux'` + "\n"
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemovePlugin(tmp, "catppuccin/tmux"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	if strings.Contains(string(data), "catppuccin/tmux") {
		t.Errorf("expected single-quoted plugin to be removed, got:\n%s", string(data))
	}
}

func TestRemovePlugin_PreservesOtherContent(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g status-right ""
set -g @plugin "catppuccin/tmux"
set -g mouse on
`
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemovePlugin(tmp, "catppuccin/tmux"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	content := string(data)

	if strings.Contains(content, "catppuccin/tmux") {
		t.Errorf("expected plugin removed, got:\n%s", content)
	}
	if !strings.Contains(content, "status-right") {
		t.Error("non-plugin content should be preserved")
	}
	if !strings.Contains(content, "mouse on") {
		t.Error("non-plugin content should be preserved")
	}
}
