package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func runUpdate(args []string) int {
	tmuxEcho := hasFlag(args, "--tmux-echo")

	// Filter out flags from args.
	var names []string
	for _, a := range args {
		if a != "--tmux-echo" && a != "--shell-echo" {
			names = append(names, a)
		}
	}

	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	// No plugin names: show interactive prompt (tmux-echo) or usage (shell).
	if len(names) == 0 {
		if tmuxEcho {
			runUpdatePrompt(runner, cfg)
			return 0
		}
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintf(os.Stderr, "  tpm update all                   update all plugins\n")
		fmt.Fprintf(os.Stderr, "  tpm update tmux-foo              update plugin 'tmux-foo'\n")
		fmt.Fprintf(os.Stderr, "  tpm update tmux-bar tmux-baz     update multiple plugins\n")
		return 1
	}

	output := newOutput(tmuxEcho, runner)

	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		output.Err("Failed to gather plugins: " + err.Error())
		return exitCode(output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	mgr.Update(ctx, plugins, names)

	if tmuxEcho {
		_ = runner.SourceFile(cfg.TmuxConf)
		output.EndMessage()
	}

	return exitCode(output)
}

// runUpdatePrompt handles the interactive update prompt from tmux keybinding.
func runUpdatePrompt(runner *tmux.RealRunner, cfg *config.Config) {
	output := ui.NewTmuxOutput(runner)

	// Reload environment.
	_ = runner.SourceFile(cfg.TmuxConf)

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		output.Err("Failed to gather plugins: " + err.Error())
		return
	}

	output.Ok("Installed plugins:")
	output.Ok("")

	mgr := newManagerDeps(cfg.PluginPath, output)

	for _, p := range plugins {
		if mgr.IsPluginInstalled(p.Name) {
			output.Ok("  " + p.Name)
		}
	}

	output.Ok("")
	output.Ok("Type plugin name to update it.")
	output.Ok("")
	output.Ok("- \"all\" - updates all plugins")
	output.Ok("- ENTER - cancels")

	binary := findBinary()
	_ = runner.CommandPrompt("plugin update:",
		"send-keys C-c; run-shell '"+binary+" update --tmux-echo %1'")
}
