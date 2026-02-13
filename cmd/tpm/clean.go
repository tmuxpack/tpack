package main

import (
	"fmt"
	"os"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func runClean(args []string) int {
	tmuxEcho := hasFlag(args, "--tmux-echo")

	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	output := newOutput(tmuxEcho, runner)

	if tmuxEcho {
		_ = runner.SourceFile(cfg.TmuxConf)
	}

	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		output.Err("Failed to gather plugins: " + err.Error())
		return exitCode(output)
	}

	mgr.Clean(plugins)

	if tmuxEcho {
		_ = runner.SourceFile(cfg.TmuxConf)
		output.EndMessage()
	}

	return exitCode(output)
}
