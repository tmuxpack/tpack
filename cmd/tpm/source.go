package main

import (
	"fmt"
	"os"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func runSource() int {
	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	output := ui.NewShellOutput()
	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	mgr.Source(plugins)
	return 0
}
