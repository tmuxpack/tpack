package main

import (
	"context"
	"fmt"
	"os"

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
		cfg, err := config.Resolve(runner)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tpack: config error:", err)
			return errSilent
		}

		output := newOutput(tmuxEcho, runner)

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
		}

		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home, xdgConfigHome(cfg.Home))

		mgr.Clean(context.Background(), plugins)

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
			output.EndMessage()
		}

		if output.HasFailed() {
			return errSilent
		}
		return nil
	},
}

func init() {
	cleanCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}
