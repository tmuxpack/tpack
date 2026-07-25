package config

import (
	"os"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/plug"
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

func TestAppendPlugin_RoundTripsSupportedSpecs(t *testing.T) {
	tests := []string{
		"catppuccin/tmux",
		"https://github.com/catppuccin/tmux.git#v2",
		"ssh://git@github.com/catppuccin/tmux.git",
		"git@github.com:catppuccin/tmux.git",
		"github.com:catppuccin/tmux.git",
		"catppuccin/tmux alias=catppuccin-theme#v2",
	}

	for _, spec := range tests {
		t.Run(spec, func(t *testing.T) {
			confPath := t.TempDir() + "/tmux.conf"
			if err := os.WriteFile(confPath, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := AppendPlugin(confPath, spec); err != nil {
				t.Fatalf("AppendPlugin: %v", err)
			}
			data, err := os.ReadFile(confPath)
			if err != nil {
				t.Fatal(err)
			}
			got := plug.ExtractPluginsFromConfig(string(data))
			if len(got) != 1 || got[0] != spec {
				t.Fatalf("round trip = %q, want [%q]", got, spec)
			}
		})
	}
}

func TestAppendPlugin_RejectsHostileSpecWithoutChangingConfig(t *testing.T) {
	confPath := t.TempDir() + "/tmux.conf"
	const original = "set -g status on\n"
	if err := os.WriteFile(confPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	hostileSpecs := []string{
		`owner/repo"; run-shell "touch /tmp/tpack-injected"; #`,
		`owner/repo#main"; run-shell "touch /tmp/tpack-injected"; #`,
		`owner/repo alias=theme";run-shell`,
	}
	for _, hostile := range hostileSpecs {
		if err := AppendPlugin(confPath, hostile); err == nil {
			t.Fatalf("AppendPlugin accepted hostile repository spec %q", hostile)
		}
	}
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("tmux.conf changed to %q", data)
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

func TestAppendPlugin_SubstringNotBlocked(t *testing.T) {
	tmp := t.TempDir() + "/tmux.conf"
	initial := `set -g @plugin "user/foobar"` + "\n"
	if err := os.WriteFile(tmp, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	// "user/foo" is a substring of "user/foobar" but should NOT be treated as duplicate.
	if err := AppendPlugin(tmp, "user/foo"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}

	data, _ := os.ReadFile(tmp)
	content := string(data)

	if !strings.Contains(content, `set -g @plugin "user/foo"`) {
		t.Errorf("expected user/foo to be added, got:\n%s", content)
	}
	if strings.Count(content, "user/foobar") != 1 {
		t.Error("original plugin should be preserved exactly once")
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
