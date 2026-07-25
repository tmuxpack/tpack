package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/tmux"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove plugin directories not declared in tmux.conf",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxEcho, _ := cmd.Flags().GetBool("tmux-echo")
		runner := tmux.NewRealRunner()
		output := newCommandOutput(tmuxEcho, runner)
		cfg, err := config.Resolve(runner)
		if err != nil {
			output.Err("config: " + err.Error())
			return outputResult(output)
		}

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
		}

		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.Paths, output.Warn)
		if err != nil {
			output.Err("config: " + err.Error())
			return outputResult(output)
		}

		mgr.Clean(context.Background(), plugins)

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
			output.EndMessage()
		}

		return outputResult(output)
	},
}

func init() {
	cleanCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}
