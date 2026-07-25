package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/manager"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all plugins declared in tmux.conf",
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		mgr.Install(ctx, plugins)

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
			output.EndMessage()
		}

		return outputResult(output)
	},
}

func init() {
	installCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}

func newCommandOutput(tmuxEcho bool, runner tmux.Runner) ui.Output {
	if tmuxEcho {
		return ui.NewReporter(ui.NewTmuxSink(runner))
	}
	return ui.NewReporter(ui.NewShellSink(os.Stdout, os.Stderr))
}

func outputResult(output ui.Output) error {
	result := output.Result()
	if result == nil {
		return nil
	}
	var transport *ui.TransportError
	if errors.As(result, &transport) {
		return result
	}
	return errSilent
}

func exitCode(output ui.Output) int {
	if output.Result() != nil {
		return 1
	}
	return 0
}

func newManagerDeps(pluginPath plug.Root, output ui.Output) *manager.Manager {
	return manager.New(pluginPath,
		gitcli.NewCloner(),
		gitcli.NewPuller(),
		gitcli.NewValidator(),
		output,
	)
}

// completePluginNames returns a list of plugin names for shell completion.
func completePluginNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.Paths, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, p := range plugins {
		names = append(names, p.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
