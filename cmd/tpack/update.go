package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/shell"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

var updateCmd = &cobra.Command{
	Use:               "update [plugin...]",
	Short:             "Update specific plugin(s) or all",
	Long:              `Update one or more plugins by name, or use "all" to update everything.`,
	ValidArgsFunction: completePluginNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxEcho, _ := cmd.Flags().GetBool("tmux-echo")
		runner := tmux.NewRealRunner()
		output := newCommandOutput(tmuxEcho, runner)
		cfg, err := config.Resolve(runner)
		if err != nil {
			output.Err("config: " + err.Error())
			return outputResult(output)
		}

		names := args

		// No plugin names: show interactive prompt (tmux-echo) or update all (shell).
		if len(names) == 0 {
			if tmuxEcho {
				runUpdatePrompt(runner, cfg, output)
				return outputResult(output)
			}
			names = []string{"all"}
		}

		mgr := newManagerDeps(cfg.PluginPath, output)

		plugins, err := loadPlugins(runner, cfg, output)
		if err != nil {
			output.Err("config: " + err.Error())
			return outputResult(output)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		mgr.Update(ctx, plugins, names)

		if tmuxEcho {
			_ = runner.SourceFile(cfg.TmuxConf)
			output.EndMessage()
		}

		return outputResult(output)
	},
}

func init() {
	updateCmd.Flags().Bool("tmux-echo", false, "output via tmux display-message")
}

// runUpdatePrompt handles the interactive update prompt from tmux keybinding.
func runUpdatePrompt(runner tmux.Runner, cfg *config.Config, output ui.Output) {
	// Reload environment.
	if err := runner.SourceFile(cfg.TmuxConf); err != nil {
		output.Err("source tmux config: " + err.Error())
		return
	}

	if !listInstalledPlugins(runner, cfg, output) {
		return
	}

	output.Ok("")
	output.Ok("Type plugin name to update it.")
	output.Ok("")
	output.Ok("- \"all\" - updates all plugins")
	output.Ok("- ENTER - cancels")

	binary := findBinary()
	if err := runner.CommandPrompt("plugin update:",
		"send-keys C-c; run-shell '"+shell.EscapeInSingleQuotes(binary)+" update --tmux-echo %1'"); err != nil {
		output.Err("open update prompt: " + err.Error())
	}
}

// listInstalledPlugins displays the list of installed plugins via output.
func listInstalledPlugins(runner tmux.Runner, cfg *config.Config, output ui.Output) bool {
	plugins, err := loadPlugins(runner, cfg, output)
	if err != nil {
		output.Err("config: " + err.Error())
		return false
	}

	output.Ok("Installed plugins:")
	output.Ok("")

	mgr := newManagerDeps(cfg.PluginPath, output)

	for _, p := range plugins {
		if mgr.IsPluginInstalled(p.DirName) {
			output.Ok("  " + p.Name)
		}
	}
	return true
}
