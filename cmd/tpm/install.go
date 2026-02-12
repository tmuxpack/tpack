package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func runInstall(args []string) int {
	tmuxEcho := hasFlag(args, "--tmux-echo")

	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	output := newOutput(tmuxEcho, runner)

	if tmuxEcho {
		// Reload tmux environment before install.
		_ = runner.SourceFile(cfg.TmuxConf)
	}

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	mgr := manager.New(cfg.PluginPath, cloner, puller, validator, output)

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		output.Err("Failed to gather plugins: " + err.Error())
		return exitCode(output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	mgr.Install(ctx, plugins)

	if tmuxEcho {
		// Reload tmux environment after install.
		_ = runner.SourceFile(cfg.TmuxConf)
		output.EndMessage()
	}

	return exitCode(output)
}

func newOutput(tmuxEcho bool, runner tmux.Runner) ui.Output {
	if tmuxEcho {
		return ui.NewTmuxOutput(runner)
	}
	return ui.NewShellOutput()
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func exitCode(output ui.Output) int {
	if output.HasFailed() {
		return 1
	}
	return 0
}
