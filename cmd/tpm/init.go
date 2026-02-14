package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/tmux-plugins/tpm/internal/config"
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
		msg := "Error, Tmux version unsupported! Please install Tmux version 1.9 or greater!"
		_ = runner.DisplayMessage(msg)
		return 1
	}

	// Resolve config.
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
		return 1
	}

	// Set TPM path in tmux environment.
	_ = runner.SetEnvironment(config.TPMEnvVar, cfg.PluginPath)

	// Find binary once and pass to functions that need it.
	binary := findBinary()

	// Bind keys.
	bindKeys(runner, cfg, binary)

	// Source plugins.
	output := ui.NewShellOutput()
	mgr := newManagerDeps(cfg.PluginPath, output)

	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	mgr.Source(plugins)

	// Spawn background update check if configured.
	if shouldSpawnUpdateCheck(cfg) {
		spawnUpdateCheck(binary)
	}

	return 0
}

func bindKeys(runner tmux.Runner, cfg *config.Config, binary string) {
	_ = runner.BindKey(cfg.InstallKey, binary+" tui --popup --install", "[tpm] Install plugins")
	_ = runner.BindKey(cfg.UpdateKey, binary+" tui --popup --update", "[tpm] Update plugins")
	_ = runner.BindKey(cfg.CleanKey, binary+" tui --popup --clean", "[tpm] Clean plugins")
}

// shouldSpawnUpdateCheck returns true if update checks are configured.
func shouldSpawnUpdateCheck(cfg *config.Config) bool {
	return cfg.UpdateMode != "" && cfg.UpdateMode != "off" && cfg.UpdateCheckInterval > 0
}

// spawnUpdateCheck launches `tpm check-updates` as a detached background process.
func spawnUpdateCheck(binary string) {
	cmd := exec.Command(binary, "check-updates") //nolint:noctx // intentionally detached, no cancellation
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "tpm: failed to spawn update check: %v\n", err)
	}
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
