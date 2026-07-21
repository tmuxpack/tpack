package main

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/parallel"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/state"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

// Update modes for the UpdateMode config setting.
const (
	updateModeOff    = "off"
	updateModePrompt = "prompt"
	updateModeAuto   = "auto"
)

var checkUpdatesCmd = &cobra.Command{
	Use:   "check-updates",
	Short: "Check if any plugins have updates available",
	RunE: func(cmd *cobra.Command, args []string) error {
		code := runCheckUpdates()
		if code != 0 {
			return errSilent
		}
		return nil
	},
}

func runCheckUpdates() int {
	runner := tmux.NewRealRunner()
	// check-updates usually runs detached from `tpack init` with stderr
	// discarded; the status line is the only channel the user sees.
	diag := ui.NewMultiOutput(ui.NewShellOutput(), ui.NewStatusOutput(runner))
	cfg, err := config.Resolve(runner)
	if err != nil {
		diag.Err("config: " + err.Error())
		return 1
	}

	if !updateChecksEnabled(cfg) {
		return 0
	}

	// Load persistent state and check interval.
	st := state.Load(cfg.StatePath, diag.Warn)
	if !st.LastUpdateCheck.IsZero() && time.Since(st.LastUpdateCheck) < cfg.UpdateCheckInterval {
		return 0
	}

	// Save timestamp before checking to prevent retry storms.
	st.LastUpdateCheck = time.Now()
	_ = state.Save(cfg.StatePath, st)

	// Gather plugins from config.
	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.Paths, diag.Warn)
	if err != nil {
		diag.Err("config: " + err.Error())
		return 1
	}

	outdated := findOutdatedPlugins(plugins, cfg.PluginPath)
	if len(outdated) == 0 {
		return 0
	}

	return handleOutdated(runner, cfg, plugins, outdated)
}

// updateChecksEnabled reports whether the update check feature is active.
func updateChecksEnabled(cfg *config.Config) bool {
	if cfg.UpdateMode == "" || cfg.UpdateMode == updateModeOff {
		return false
	}
	return cfg.UpdateCheckInterval > 0
}

const maxConcurrentChecks = 5

// findOutdatedPlugins checks each installed plugin for available updates in parallel.
func findOutdatedPlugins(plugins []plug.Plugin, pluginPath plug.Root) []string {
	validator := gitcli.NewValidator()
	fetcher := gitcli.NewFetcher()

	type target struct {
		name string
		dir  string
	}

	var targets []target
	for _, p := range plugins {
		dir, err := pluginPath.Child(p.Name)
		if err != nil {
			continue
		}
		if validator.IsGitRepo(dir) {
			targets = append(targets, target{name: p.Name, dir: dir})
		}
	}

	var (
		mu       sync.Mutex
		outdated []string
	)

	parallel.Do(targets, maxConcurrentChecks, func(t target) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		isOutdated, err := fetcher.IsOutdated(ctx, t.dir)
		if err != nil || !isOutdated {
			return
		}

		mu.Lock()
		outdated = append(outdated, t.name)
		mu.Unlock()
	})

	return outdated
}

// handleOutdated acts on the list of outdated plugins based on the configured update mode.
func handleOutdated(runner tmux.Runner, cfg *config.Config, plugins []plug.Plugin, outdated []string) int {
	switch cfg.UpdateMode {
	case updateModePrompt:
		status := ui.NewStatusOutput(runner)
		status.Ok(strconv.Itoa(len(outdated)) + " plugin update(s) available. Press prefix+U to update.")

	case updateModeAuto:
		return autoUpdatePlugins(runner, cfg, plugins, outdated)
	}

	return 0
}

// autoUpdatePlugins performs automatic updates for the given outdated plugins.
func autoUpdatePlugins(runner tmux.Runner, cfg *config.Config, plugins []plug.Plugin, outdated []string) int {
	output := newOutput(false, runner)
	mgr := newManagerDeps(cfg.PluginPath, output)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	mgr.Update(ctx, plugins, outdated)

	status := ui.NewStatusOutput(runner)
	if output.HasFailed() {
		status.Err("auto-update failed for some plugins: " + strings.Join(outdated, ", "))
		return 1
	}
	status.Ok(strconv.Itoa(len(outdated)) + " plugin(s) updated successfully.")
	return 0
}
