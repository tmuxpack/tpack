package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

// Loading point for plugins

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Source all plugins without installing",
	RunE: func(cmd *cobra.Command, args []string) error {
		runner := tmux.NewRealRunner()
		cfg, err := config.Resolve(runner)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tpack: config error:", err)
			return errSilent
		}

		output := ui.NewShellOutput()
		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home, xdgConfigHome(cfg.Home))

		mgr.Source(context.Background(), plugins)
		return nil
	},
}
