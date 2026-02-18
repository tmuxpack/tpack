package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/parallel"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/state"
	"github.com/tmuxpack/tpack/internal/tmux"
)

func runCheckUpdates() int {
	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: config error:", err)
		return 1
	}

	if !updateChecksEnabled(cfg) {
		return 0
	}

	// Load persistent state and check interval.
	st := state.Load(cfg.StatePath)
	if !st.LastUpdateCheck.IsZero() && time.Since(st.LastUpdateCheck) < cfg.UpdateCheckInterval {
		return 0
	}

	// Save timestamp before checking to prevent retry storms.
	st.LastUpdateCheck = time.Now()
	_ = state.Save(cfg.StatePath, st)

	// Gather plugins from config.
	plugins := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, cfg.Home)

	outdated := findOutdatedPlugins(plugins, cfg.PluginPath)
	if len(outdated) == 0 {
		return 0
	}

	return handleOutdated(runner, cfg, plugins, outdated)
}

// updateChecksEnabled reports whether the update check feature is active.
func updateChecksEnabled(cfg *config.Config) bool {
	if cfg.UpdateMode == "" || cfg.UpdateMode == "off" {
		return false
	}
	return cfg.UpdateCheckInterval > 0
}

const maxConcurrentChecks = 5

// findOutdatedPlugins checks each installed plugin for available updates in parallel.
func findOutdatedPlugins(plugins []plug.Plugin, pluginPath string) []string {
	validator := gitcli.NewValidator()
	fetcher := gitcli.NewFetcher()

	type target struct {
		name string
		dir  string
	}

	var targets []target
	for _, p := range plugins {
		dir := plug.PluginPath(p.Name, pluginPath)
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
func handleOutdated(runner *tmux.RealRunner, cfg *config.Config, plugins []plug.Plugin, outdated []string) int {
	switch cfg.UpdateMode {
	case "prompt":
		msg := "tpack: " + strconv.Itoa(len(outdated)) + " plugin update(s) available. Press prefix+U to update."
		_ = runner.DisplayMessage(msg)

	case "auto":
		return autoUpdatePlugins(runner, cfg, plugins, outdated)
	}

	return 0
}

// autoUpdatePlugins performs automatic updates for the given outdated plugins.
func autoUpdatePlugins(runner *tmux.RealRunner, cfg *config.Config, plugins []plug.Plugin, outdated []string) int {
	output := newOutput(false, runner)
	mgr := newManagerDeps(cfg.PluginPath, output)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	mgr.Update(ctx, plugins, outdated)

	if output.HasFailed() {
		_ = runner.DisplayMessage("tpack: auto-update failed for some plugins: " + strings.Join(outdated, ", "))
		return 1
	}
	_ = runner.DisplayMessage("tpack: " + strconv.Itoa(len(outdated)) + " plugin(s) updated successfully.")
	return 0
}
