package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

func runSource() int {
	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: config error:", err)
		return 1
	}

	output := ui.NewShellOutput()
	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	mgr.Source(context.Background(), plugins)
	return 0
}
