package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/shell"
	"github.com/tmuxpack/tpack/internal/tmux"
)

type hiddenEnvironmentFS struct {
	config.RealFS
}

func (hiddenEnvironmentFS) ReadFile(name string) ([]byte, error) {
	if name == "/etc/tmux.conf" {
		return nil, os.ErrNotExist
	}
	return config.RealFS{}.ReadFile(name)
}

func TestLoadSourceGraphExpandsHiddenEnvironment(t *testing.T) {
	tmuxBin, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	versionCommand := exec.CommandContext(ctx, tmuxBin, "-V")
	versionOutput, err := versionCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("tmux version: %v\n%s", err, versionOutput)
	}
	if tmux.ParseVersionDigits(string(versionOutput)) < 302 {
		t.Skipf("%%hidden requires tmux 3.2+, got %s", strings.TrimSpace(string(versionOutput)))
	}

	home := filepath.Join(t.TempDir(), "home with ' quote")
	hiddenDir := filepath.Join(home, "hidden")
	rootConfig := filepath.Join(home, "tmux.conf")
	sourcedConfig := filepath.Join(hiddenDir, "plugins.conf")
	if mkdirErr := os.MkdirAll(hiddenDir, 0o755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	if writeErr := os.WriteFile(sourcedConfig, []byte("set -g @plugin owner/hidden\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	rootContent := "%hidden TPACK_HIDDEN_CONFIG=" + shell.Quote(hiddenDir) + "\n" +
		"source-file \"$TPACK_HIDDEN_CONFIG/plugins.conf\"\n"
	if writeErr := os.WriteFile(rootConfig, []byte(rootContent), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	t.Setenv("HOME", home)
	t.Setenv("TPACK_HIDDEN_CONFIG", "")
	serverEnv := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || name == "TMUX" || name == "TMUX_PANE" || name == "TPACK_HIDDEN_CONFIG" {
			continue
		}
		serverEnv = append(serverEnv, entry)
	}

	socket := fmt.Sprintf("tpack-hidden-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		kill := exec.CommandContext(cleanupCtx, tmuxBin, "-L", socket, "kill-server")
		kill.Env = serverEnv
		_ = kill.Run()
	})
	start := exec.CommandContext(ctx, tmuxBin, "-L", socket, "-f", rootConfig, "new-session", "-d")
	start.Env = serverEnv
	if output, startErr := start.CombinedOutput(); startErr != nil {
		t.Fatalf("start tmux: %v\n%s", startErr, output)
	}

	tmuxValue := exec.CommandContext(ctx, tmuxBin, "-L", socket, "display-message", "-p", "#{socket_path},#{pid},#{session_id}")
	tmuxValue.Env = serverEnv
	tmuxOutput, err := tmuxValue.CombinedOutput()
	if err != nil {
		t.Fatalf("read TMUX value: %v\n%s", err, tmuxOutput)
	}
	t.Setenv("TMUX", strings.TrimSpace(string(tmuxOutput)))
	paneValue := exec.CommandContext(ctx, tmuxBin, "-L", socket, "display-message", "-p", "#{pane_id}")
	paneValue.Env = serverEnv
	paneOutput, err := paneValue.CombinedOutput()
	if err != nil {
		t.Fatalf("read TMUX_PANE value: %v\n%s", err, paneOutput)
	}
	t.Setenv("TMUX_PANE", strings.TrimSpace(string(paneOutput)))

	graph, err := config.LoadSourceGraph(tmux.NewRealRunner(), hiddenEnvironmentFS{}, config.Paths{
		TmuxConf:      rootConfig,
		TmuxConfs:     []string{rootConfig},
		Home:          home,
		XDGConfigHome: filepath.Join(home, ".config"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{rootConfig, sourcedConfig}; !reflect.DeepEqual(graph.Paths(), want) {
		t.Fatalf("source graph paths = %v, want %v", graph.Paths(), want)
	}
}
