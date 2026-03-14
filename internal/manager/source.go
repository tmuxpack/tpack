package manager

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tmuxpack/tpack/internal/plug"
)

// Source executes all *.tmux files from each plugin directory.
func (m *Manager) Source(ctx context.Context, plugins []plug.Plugin) {
	for _, p := range plugins {
		dir := plug.PluginPath(p.Name, m.pluginPath)
		m.sourcePlugin(ctx, dir)
	}
}

func (m *Manager) sourcePlugin(ctx context.Context, dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmux"))
	if err != nil {
		m.output.Err("glob error for " + dir + ": " + err.Error())
		return
	}

	for _, file := range matches {
		cmd := exec.CommandContext(ctx, file) //nolint:gosec // plugin files are user-configured
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			// Retry with the interpreter from the shebang looked up via
			// PATH if the absolute interpreter path was not found (e.g.
			// Termux where /usr/bin/env does not exist).
			if errors.Is(err, syscall.ENOENT) {
				if interp := parseShebangInterpreter(file); interp != "" {
					fallback := exec.CommandContext(ctx, interp, file) //nolint:gosec // plugin files are user-configured
					fallback.Stdout = io.Discard
					fallback.Stderr = io.Discard
					err = fallback.Run()
				}
			}
			if err != nil {
				m.output.Err("error sourcing " + filepath.Base(file) + ": " + err.Error())
			}
		}
	}
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
