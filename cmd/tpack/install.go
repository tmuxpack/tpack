package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all plugins declared in tmux.conf",
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		mgr.Install(ctx, plugins)

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
	installCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}

func newOutput(tmuxEcho bool, runner tmux.Runner) ui.Output {
	if tmuxEcho {
		return ui.NewTmuxOutput(runner)
	}
	return ui.NewShellOutput()
}

func exitCode(output ui.Output) int {
	if output.HasFailed() {
		return 1
	}
	return 0
}

func xdgConfigHome(home string) string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(home, ".config")
}

func newManagerDeps(pluginPath string, output ui.Output) *manager.Manager {
	return manager.New(pluginPath,
		gitcli.NewCloner(),
		gitcli.NewPuller(),
		gitcli.NewValidator(),
		output,
	)
}
