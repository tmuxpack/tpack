package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/tui"
)

func runTui(args []string) int {
	popup := hasFlag(args, "--popup")

	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: failed to gather plugins:", err)
		return 1
	}

	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()

	if popup {
		return launchPopup(cfg, plugins, cloner, puller, validator)
	}

	if err := tui.Run(cfg, plugins, cloner, puller, validator); err != nil {
		fmt.Fprintln(os.Stderr, "tpm:", err)
		return 1
	}
	return 0
}

func launchPopup(
	cfg *config.Config,
	plugins []plugin.Plugin,
	cloner git.Cloner,
	puller git.Puller,
	validator git.Validator,
) int {
	w, h := tui.IdealSize(cfg, plugins, cloner, puller, validator)

	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: cannot find executable:", err)
		return 1
	}

	cmd := exec.CommandContext(context.Background(), "tmux", "display-popup",
		"-E",
		"-w", fmt.Sprintf("%d", w),
		"-h", fmt.Sprintf("%d", h),
		binary+" tui",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tpm: popup failed:", err)
		return 1
	}
	return 0
}
