package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/state"
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

		plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.Paths)

		failures := mgr.Source(context.Background(), plugins)
		for _, f := range failures {
			output.Err("error loading " + f.Name + ": " + f.Message)
		}
		if err := state.SaveLoadErrors(cfg.StatePath, failures); err != nil {
			fmt.Fprintf(os.Stderr, "tpack: warning: failed to save load errors: %v\n", err)
		}
		return nil
	},
}
