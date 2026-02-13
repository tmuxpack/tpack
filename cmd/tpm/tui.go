package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/tui"
)

func runTui(args []string) int {
	popup := hasFlag(args, "--popup")

	// Parse operation flags.
	var autoOp tui.Operation
	switch {
	case hasFlag(args, "--install"):
		autoOp = tui.OpInstall
	case hasFlag(args, "--update"):
		autoOp = tui.OpUpdate
	case hasFlag(args, "--clean"):
		autoOp = tui.OpClean
	}

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

	deps := tui.Deps{
		Cloner:    git.NewCLICloner(),
		Puller:    git.NewCLIPuller(),
		Validator: git.NewCLIValidator(),
		Fetcher:   git.NewCLIFetcher(),
		RevParser: git.NewCLIRevParser(),
		Logger:    git.NewCLILogger(),
	}
	if autoOp == tui.OpInstall || autoOp == tui.OpUpdate {
		deps.Runner = runner
	}

	var opts []tui.ModelOption
	if autoOp != tui.OpNone {
		opts = append(opts, tui.WithAutoOp(autoOp))
	}

	if popup {
		return launchPopup(cfg, plugins, deps, opts, autoOp)
	}

	if err := tui.Run(cfg, plugins, deps, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "tpm:", err)
		return 1
	}
	return 0
}

func launchPopup(
	cfg *config.Config,
	plugins []plugin.Plugin,
	deps tui.Deps,
	opts []tui.ModelOption,
	autoOp tui.Operation,
) int {
	w, h := tui.IdealSize(cfg, plugins, deps, opts...)

	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: cannot find executable:", err)
		return 1
	}

	// Build the subprocess command.
	subcmd := shellescape(binary) + " tui"
	if autoOp != tui.OpNone {
		subcmd += " --" + strings.ToLower(autoOp.String())
	}

	cmd := exec.CommandContext(context.Background(), "tmux", "display-popup",
		"-E",
		"-w", fmt.Sprintf("%d", w),
		"-h", fmt.Sprintf("%d", h),
		subcmd,
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

// shellescape wraps s in single quotes, escaping any embedded single quotes.
func shellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
