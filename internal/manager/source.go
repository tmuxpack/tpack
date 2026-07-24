package manager

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tmuxpack/tpack/internal/plug"
)

// Source executes every plugin's *.tmux files and returns one LoadFailure per
// failing plugin (messages aggregated across that plugin's files).
func (m *Manager) Source(ctx context.Context, plugins []plug.Plugin) []plug.LoadFailure {
	var failures []plug.LoadFailure
	for _, p := range plugins {
		dir, err := m.pluginRoot.Child(p.DirName)
		if err != nil {
			failures = append(failures, plug.LoadFailure{Name: p.Name, DirName: p.DirName, Message: err.Error()})
			continue
		}
		if msg := sourcePlugin(ctx, dir); msg != "" {
			failures = append(failures, plug.LoadFailure{Name: p.Name, DirName: p.DirName, Message: msg})
		}
	}
	return failures
}

// sourcePlugin executes all *.tmux files in dir and returns an aggregated
// error message describing any failures ("" when all succeed or dir is absent).
func sourcePlugin(ctx context.Context, dir string) string {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmux"))
	if err != nil {
		return "glob error for " + dir + ": " + err.Error()
	}

	var msgs []string
	for _, file := range matches {
		out, err := runTmuxFile(ctx, file)
		// Retry via the shebang interpreter looked up on PATH when the
		// absolute interpreter path is missing (e.g. Termux has no /usr/bin/env).
		if errors.Is(err, syscall.ENOENT) {
			if interp := parseShebangInterpreter(file); interp != "" {
				out, err = runTmuxFile(ctx, interp, file)
			}
		}
		// Retry through a shell if the file has no shebang and is not a
		// binary (ENOEXEC). TPM sources *.tmux files via run-shell, so
		// shebang-less plugin scripts work there; match that behavior.
		if errors.Is(err, syscall.ENOEXEC) {
			out, err = runTmuxFile(ctx, "sh", file)
		}
		if err != nil {
			msg := "error sourcing " + filepath.Base(file) + ": " + err.Error()
			// Include the plugin's own output, not just the exit status.
			if detail := strings.TrimSpace(out); detail != "" {
				msg += "\n" + indentLines(detail)
			}
			msgs = append(msgs, msg)
		}
	}
	return strings.Join(msgs, "\n")
}

// runTmuxFile executes a plugin entry script (optionally via an explicit
// interpreter) and returns its combined stdout and stderr output.
func runTmuxFile(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // plugin files are user-configured
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// indentLines prefixes every line with two spaces.
func indentLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

// parseShebangInterpreter reads the shebang line from a script and returns
// the interpreter base name (e.g. "bash" from "#!/usr/bin/env bash" or
// "#!/bin/bash"). Returns "" if no shebang is found.
func parseShebangInterpreter(path string) string {
	f, err := os.Open(path) //nolint:gosec // path comes from filepath.Glob on user-configured plugin dirs
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "#!") {
		return ""
	}
	line = strings.TrimPrefix(line, "#!")
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}

	// "#!/usr/bin/env bash" → interpreter is the second field.
	if filepath.Base(fields[0]) == "env" && len(fields) > 1 {
		return fields[1]
	}
	// "#!/bin/bash" → interpreter is the base name.
	return filepath.Base(fields[0])
}
