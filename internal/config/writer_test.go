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
