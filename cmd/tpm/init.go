package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/manager"
	"github.com/tmux-plugins/tpm/internal/tmux"
	"github.com/tmux-plugins/tpm/internal/ui"
)

func runInit() int {
	runner := tmux.NewRealRunner()

	// Check tmux version.
	verStr, err := runner.Version()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: failed to get tmux version")
		return 1
	}
	current := tmux.ParseVersionDigits(verStr)
	if !tmux.IsVersionSupported(current, config.SupportedTmuxVersion) {
		msg := fmt.Sprintf("Error, Tmux version unsupported! Please install Tmux version 1.9 or greater!")
		runner.DisplayMessage(msg)
		return 1
	}

	// Resolve config.
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	// Set TPM path in tmux environment.
	runner.SetEnvironment(config.TPMEnvVar, cfg.PluginPath)

	// Bind keys.
	bindKeys(runner, cfg)

	// Source plugins.
	cloner := git.NewCLICloner()
	puller := git.NewCLIPuller()
	validator := git.NewCLIValidator()
	output := ui.NewShellOutput()
	mgr := manager.New(cfg.PluginPath, cloner, puller, validator, output)

	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: failed to gather plugins:", err)
		return 1
	}

	mgr.Source(plugins)
	return 0
}

func bindKeys(runner tmux.Runner, cfg *config.Config) {
	// Find the Go binary path for self-referencing.
	binary := findBinary()

	runner.BindKey(cfg.InstallKey, binary+" install --tmux-echo")
	runner.BindKey(cfg.UpdateKey, binary+" update --tmux-echo")
	runner.BindKey(cfg.CleanKey, binary+" clean --tmux-echo")
}

// findBinary returns the absolute path to the tpm-go binary.
func findBinary() string {
	// Try the executable path first.
	exe, err := os.Executable()
	if err == nil {
		resolved, err := filepath.EvalSymlinks(exe)
		if err == nil {
			return resolved
		}
		return exe
	}
	// Fallback: try to find alongside the tpm script.
	if dir := os.Getenv("CURRENT_DIR"); dir != "" {
		candidate := filepath.Join(dir, "tpm-go")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "tpm-go"
}
