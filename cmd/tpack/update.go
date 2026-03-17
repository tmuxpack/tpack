package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/shell"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

var updateCmd = &cobra.Command{
	Use:   "update [plugin...]",
	Short: "Update specific plugin(s) or all",
	Long:  `Update one or more plugins by name, or use "all" to update everything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxEcho, _ := cmd.Flags().GetBool("tmux-echo")

		runner := tmux.NewRealRunner()
		cfg, err := config.Resolve(runner)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tpack: config error:", err)
			return errSilent
		}

		names := args

		// No plugin names: show interactive prompt (tmux-echo) or update all (shell).
		if len(names) == 0 {
			if tmuxEcho {
				runUpdatePrompt(runner, cfg)
				return nil
			}
			names = []string{"all"}
		}

		output := newOutput(tmuxEcho, runner)

		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home, xdgConfigHome(cfg.Home))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		mgr.Update(ctx, plugins, names)

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
	updateCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}

// runUpdatePrompt handles the interactive update prompt from tmux keybinding.
func runUpdatePrompt(runner *tmux.RealRunner, cfg *config.Config) {
	output := ui.NewTmuxOutput(runner)

	// Reload environment.
	_ = runner.SourceFile(cfg.TmuxConf)

	listInstalledPlugins(runner, cfg, output)

	output.Ok("")
	output.Ok("Type plugin name to update it.")
	output.Ok("")
	output.Ok("- \"all\" - updates all plugins")
	output.Ok("- ENTER - cancels")

	binary := findBinary()
	_ = runner.CommandPrompt("plugin update:",
		"send-keys C-c; run-shell '"+shell.EscapeInSingleQuotes(binary)+" update --tmux-echo %1'")
}

// listInstalledPlugins displays the list of installed plugins via output.
func listInstalledPlugins(runner *tmux.RealRunner, cfg *config.Config, output ui.Output) {
	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home, xdgConfigHome(cfg.Home))

	output.Ok("Installed plugins:")
	output.Ok("")

	mgr := newManagerDeps(cfg.PluginPath, output)

	for _, p := range plugins {
		if mgr.IsPluginInstalled(p.Name) {
			output.Ok("  " + p.Name)
		}
	}
}
