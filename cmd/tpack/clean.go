package main

import (
	"fmt"
	"os"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func runClean(args []string) int {
	tmuxEcho := hasFlag(args, "--tmux-echo")

	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: config error:", err)
		return 1
	}

	output := newOutput(tmuxEcho, runner)

	if tmuxEcho {
		_ = runner.SourceFile(cfg.TmuxConf)
	}

	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	mgr.Clean(plugins)

	if tmuxEcho {
		_ = runner.SourceFile(cfg.TmuxConf)
		output.EndMessage()
	}

	return exitCode(output)
}
