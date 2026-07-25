package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmuxpack/tpack/internal/plug"
)

func isPluginManagerInit(line string) bool {
	args, ok := splitTmuxCommandLine(line)
	if !ok || len(args) < 2 || (args[0] != "run" && args[0] != "run-shell") {
		return false
	}

	commandAt, ok := runShellCommandIndex(args)
	if !ok || commandAt >= len(args) {
		return false
	}
	command := args[commandAt:]
	return strings.Join(command, " ") == "tpack init" ||
		(len(command) == 1 && !strings.ContainsAny(command[0], " \t") && strings.HasSuffix(command[0], "/tpm"))
}

func splitTmuxCommandLine(line string) ([]string, bool) {
	var args []string
	var word strings.Builder
	var quote byte
	inWord := false

	flush := func() {
		if inWord {
			args = append(args, word.String())
			word.Reset()
			inWord = false
		}
	}

	for i := 0; i < len(line); i++ {
		char := line[i]
		if quote != 0 {
			if consumeQuotedTmuxChar(line, &i, quote, &word) {
				quote = 0
			}
			continue
		}

		switch {
		case char == '\'' || char == '"':
			quote = char
			inWord = true
		case char == '#' && !inWord:
			flush()
			return args, true
		case char == '\\' && i+1 < len(line):
			inWord = true
			i++
			word.WriteByte(line[i])
		case isTmuxWhitespace(char):
			flush()
		default:
			inWord = true
			word.WriteByte(char)
		}
	}
	if quote != 0 {
		return nil, false
	}
	flush()
	return args, true
}

func consumeQuotedTmuxChar(line string, at *int, quote byte, word *strings.Builder) bool {
	char := line[*at]
	if char == quote {
		return true
	}
	if char == '\\' && quote == '"' && *at+1 < len(line) {
		*at++
		word.WriteByte(line[*at])
		return false
	}
	word.WriteByte(char)
	return false
}

func isTmuxWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

func runShellCommandIndex(args []string) (int, bool) {
	for i := 1; i < len(args); i++ {
		option := args[i]
		if option == "--" {
			return i + 1, true
		}
		if len(option) < 2 || option[0] != '-' {
			return i, true
		}

		for optionAt := 1; optionAt < len(option); optionAt++ {
			switch option[optionAt] {
			case 'b', 'C':
			case 'd', 't':
				if optionAt+1 < len(option) {
					optionAt = len(option)
					continue
				}
				i++
				if i >= len(args) {
					return 0, false
				}
			case '-':
				return 0, false
			default:
				return 0, false
			}
		}
	}
	return len(args), true
}

// Adds a `set -g @plugin "repo"` line to the tmux.conf file if not already there.
func AppendPlugin(confPath string, repo string) error {
	if _, err := plug.ParseSpec(repo, nil); err != nil {
		return fmt.Errorf("validate plugin: %w", err)
	}
	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read tmux.conf: %w", err)
	}

	content := string(data)
	lastPluginEnd := -1
	firstInitStart := -1
	offset := 0
	for segment := range strings.SplitAfterSeq(content, "\n") {
		if plug.MatchesPluginLine(segment, repo) {
			return nil
		}
		if len(plug.ExtractPluginsFromConfig(segment)) > 0 {
			lastPluginEnd = offset + len(segment)
		}
		if firstInitStart < 0 && isPluginManagerInit(segment) {
			firstInitStart = offset
		}
		offset += len(segment)
	}

	insertAt := len(content)
	if lastPluginEnd >= 0 {
		insertAt = lastPluginEnd
	} else if firstInitStart >= 0 {
		insertAt = firstInitStart
	}

	line := fmt.Sprintf("set -g @plugin %q\n", repo)
	if insertAt > 0 && content[insertAt-1] != '\n' {
		line = "\n" + line
	}
	updated := content[:insertAt] + line + content[insertAt:]

	return rewriteConfigAtomically(confPath, updated)
}

func rewriteConfigAtomically(confPath string, content string) (returnErr error) {
	targetPath, err := filepath.EvalSymlinks(confPath)
	if err != nil {
		return fmt.Errorf("resolve tmux.conf target: %w", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("stat tmux.conf target: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary tmux.conf: %w", err)
	}
	temporaryPath := temporary.Name()
	temporaryOpen := true
	defer func() {
		if temporaryOpen {
			if closeErr := temporary.Close(); closeErr != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("close temporary tmux.conf: %w", closeErr))
			}
		}
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
			returnErr = errors.Join(returnErr, fmt.Errorf("clean up temporary tmux.conf: %w", removeErr))
		}
	}()

	if _, err := temporary.WriteString(content); err != nil {
		return fmt.Errorf("write temporary tmux.conf: %w", err)
	}
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("set temporary tmux.conf permissions: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary tmux.conf: %w", err)
	}
	if err := temporary.Close(); err != nil {
		temporaryOpen = false
		return fmt.Errorf("close temporary tmux.conf: %w", err)
	}
	temporaryOpen = false
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		return fmt.Errorf("rename temporary tmux.conf: %w", err)
	}
	return nil
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

	return os.WriteFile(confPath, []byte(strings.Join(kept, "\n")), 0o600) //nolint:gosec // confPath is resolved from user config
}
