package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/shell"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/tui"
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

	// Fall back to inline TUI if tmux doesn't support display-popup (< 3.2).
	if popup {
		if verStr, err := runner.Version(); err != nil || !popupSupported(verStr) {
			popup = false
		}
	}

	theme := tui.BuildTheme(runner)
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: config error:", err)
		return 1
	}
	theme = tui.OverlayConfigColors(theme, cfg.Colors)

	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	deps := tui.Deps{
		Cloner:    gitcli.NewCloner(),
		Puller:    gitcli.NewPuller(),
		Validator: gitcli.NewValidator(),
		Fetcher:   gitcli.NewFetcher(),
		RevParser: gitcli.NewRevParser(),
		Logger:    gitcli.NewLogger(),
	}
	deps.Runner = runner

	var opts []tui.ModelOption
	opts = append(opts, tui.WithTheme(theme))
	opts = append(opts, tui.WithVersion(version))
	opts = append(opts, tui.WithBinaryPath(findBinary()))
	if autoOp != tui.OpNone {
		opts = append(opts, tui.WithAutoOp(autoOp))
	}

	if popup {
		return launchPopup(cfg, plugins, deps, opts, autoOp)
	}

	if err := tui.Run(cfg, plugins, deps, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "tpack:", err)
		return 1
	}
	return 0
}

func launchPopup(
	cfg *config.Config,
	plugins []plug.Plugin,
	deps tui.Deps,
	opts []tui.ModelOption,
	autoOp tui.Operation,
) int {
	w, h := tui.IdealSize(cfg, plugins, deps, opts...)

	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: cannot find executable:", err)
		return 1
	}

	// Build the subprocess command.
	subcmd := shell.Quote(binary) + " tui"
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
		// Popup failed (e.g. terminal too small); fall back to inline TUI.
		if err := tui.Run(cfg, plugins, deps, opts...); err != nil {
			fmt.Fprintln(os.Stderr, "tpack:", err)
			return 1
		}
	}
	return 0
}

// popupMinVersion is the minimum tmux version that supports display-popup,
// encoded as major*100 + minor (tmux 3.2).
const popupMinVersion = 302

// popupSupported reports whether the given tmux version string supports
// display-popup (introduced in tmux 3.2).
func popupSupported(versionStr string) bool {
	return tmux.ParseVersionDigits(versionStr) >= popupMinVersion
}
