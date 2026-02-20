package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
)

// Adds a `set -g @plugin "repo"` line to the tmux.conf file if not already there.
// TODO: Should find where other plugins are and inserted near them
func AppendPlugin(confPath string, repo string) error {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read tmux.conf: %w", err)
	}

	content := string(data)
	if strings.Contains(content, repo) {
		return nil
	}

	line := fmt.Sprintf("set -g @plugin \"%s\"\n", repo)

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		line = "\n" + line
	}

	f, err := os.OpenFile(confPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open tmux.conf: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(line)
	return err
}

// removes plugin from tmux.conf if found
func RemovePlugin(confPath string, spec string) error {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read tmux.conf: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var kept []string
	found := false
	for _, line := range lines {
		if plug.MatchesPluginLine(line, spec) {
			found = true
			continue
		}
		kept = append(kept, line)
	}

	if !found {
		return nil
	}

	return os.WriteFile(confPath, []byte(strings.Join(kept, "\n")), 0o600)
}
