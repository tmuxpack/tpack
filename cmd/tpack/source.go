package main

import (
	"context"

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
		output := ui.NewShellOutput()
		cfg, err := config.Resolve(runner)
		if err != nil {
			output.Err("config: " + err.Error())
			return errSilent
		}

		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.Paths, output.Warn)
		if err != nil {
			output.Err("config: " + err.Error())
			return errSilent
		}

		failures := mgr.Source(context.Background(), plugins)
		for _, f := range failures {
			output.Err("error loading " + f.Name + ": " + f.Message)
		}
		if err := state.SaveLoadErrors(cfg.StatePath, failures); err != nil {
			output.Warn("failed to save load errors: " + err.Error())
		}
		return nil
	},
}
