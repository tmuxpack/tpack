package config

import (
	"os"
	"path/filepath"
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

func TestAppendPlugin_Placement(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		want    string
	}{
		{
			name: "after existing plugin and before tpack init",
			initial: "set -g @plugin 'tmux-plugins/tmux-resurrect'\n\n" +
				"run 'tpack init'\n",
			want: "set -g @plugin 'tmux-plugins/tmux-resurrect'\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n\n" +
				"run 'tpack init'\n",
		},
		{
			name: "after last supported plugin declaration",
			initial: "set-option -g @plugin 'owner/one'\n" +
				"set -g @plugin owner/two\n\nset -g mouse on\n",
			want: "set-option -g @plugin 'owner/one'\n" +
				"set -g @plugin owner/two\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n\nset -g mouse on\n",
		},
		{
			name:    "before tpack init without existing plugins",
			initial: "set -g mouse on\nrun-shell \"tpack init\"\n",
			want: "set -g mouse on\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n" +
				"run-shell \"tpack init\"\n",
		},
		{
			name:    "before TPM init without existing plugins",
			initial: "run '~/.tmux/plugins/tpm/tpm'\n",
			want: "set -g @plugin \"Ataraxy-Labs/opensessions\"\n" +
				"run '~/.tmux/plugins/tpm/tpm'\n",
		},
		{
			name:    "before background tpack init",
			initial: "set -g mouse on\nrun-shell -b \"tpack init\"\n",
			want: "set -g mouse on\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n" +
				"run-shell -b \"tpack init\"\n",
		},
		{
			name:    "before tpack init with trailing comment",
			initial: "set -g mouse on\nrun \"tpack init\" # initialize\n",
			want: "set -g mouse on\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n" +
				"run \"tpack init\" # initialize\n",
		},
		{
			name:    "append after non-init command ending in TPM text",
			initial: "run-shell \"echo /tpm\"\nset -g mouse on\n",
			want: "run-shell \"echo /tpm\"\nset -g mouse on\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n",
		},
		{
			name:    "append when only commented anchors exist",
			initial: "# set -g @plugin 'owner/disabled'\n# run 'tpack init'\nset -g mouse on\n",
			want: "# set -g @plugin 'owner/disabled'\n# run 'tpack init'\nset -g mouse on\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n",
		},
		{
			name:    "add newline after unterminated plugin declaration",
			initial: "set -g @plugin 'owner/one'",
			want: "set -g @plugin 'owner/one'\n" +
				"set -g @plugin \"Ataraxy-Labs/opensessions\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confPath := t.TempDir() + "/tmux.conf"
			if err := os.WriteFile(confPath, []byte(tt.initial), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := AppendPlugin(confPath, "Ataraxy-Labs/opensessions"); err != nil {
				t.Fatalf("AppendPlugin: %v", err)
			}
			got, err := os.ReadFile(confPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("tmux.conf = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppendPlugin_AtomicallyReplacesConfig(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "tmux.conf")
	linkedPath := filepath.Join(dir, "original.conf")
	const initial = "set -g mouse on\n"
	if err := os.WriteFile(confPath, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(confPath, linkedPath); err != nil {
		t.Fatal(err)
	}

	if err := AppendPlugin(confPath, "Ataraxy-Labs/opensessions"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}
	linked, err := os.ReadFile(linkedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(linked) != initial {
		t.Fatalf("original file content = %q, want %q", linked, initial)
	}
}

func TestAppendPlugin_PreservesPermissions(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "tmux.conf")
	if err := os.WriteFile(confPath, []byte("set -g mouse on\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := AppendPlugin(confPath, "Ataraxy-Labs/opensessions"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}
	info, err := os.Stat(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Fatalf("tmux.conf permissions = %o, want %o", got, want)
	}
}

func TestAppendPlugin_ReplacesSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "actual.conf")
	confPath := filepath.Join(dir, "tmux.conf")
	if err := os.WriteFile(targetPath, []byte("set -g mouse on\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(targetPath), confPath); err != nil {
		t.Fatal(err)
	}

	if err := AppendPlugin(confPath, "Ataraxy-Labs/opensessions"); err != nil {
		t.Fatalf("AppendPlugin: %v", err)
	}
	info, err := os.Lstat(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("tmux.conf symlink was replaced")
	}
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `set -g @plugin "Ataraxy-Labs/opensessions"`) {
		t.Fatalf("symlink target was not updated: %q", got)
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
