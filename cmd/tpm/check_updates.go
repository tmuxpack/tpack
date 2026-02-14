package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/state"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func runCheckUpdates() int {
	runner := tmux.NewRealRunner()
	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: config error:", err)
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
	plugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpm: failed to gather plugins:", err)
		return 1
	}

	outdated := findOutdatedPlugins(plugins, cfg.PluginPath)
	if len(outdated) == 0 {
		return 0
	}

	return handleOutdated(runner, cfg, outdated)
}

// updateChecksEnabled reports whether the update check feature is active.
func updateChecksEnabled(cfg *config.Config) bool {
	if cfg.UpdateMode == "" || cfg.UpdateMode == "off" {
		return false
	}
	return cfg.UpdateCheckInterval > 0
}

// findOutdatedPlugins checks each installed plugin for available updates in parallel.
func findOutdatedPlugins(plugins []plugin.Plugin, pluginPath string) []string {
	validator := git.NewCLIValidator()
	fetcher := git.NewCLIFetcher()

	var (
		mu       sync.Mutex
		outdated []string
		wg       sync.WaitGroup
	)

	for _, p := range plugins {
		dir := plugin.PluginPath(p.Name, pluginPath)
		if !validator.IsGitRepo(dir) {
			continue
		}

		wg.Add(1)
		go func(name, dir string) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			isOutdated, err := fetcher.IsOutdated(ctx, dir)
			if err != nil || !isOutdated {
				return
			}

			mu.Lock()
			outdated = append(outdated, name)
			mu.Unlock()
		}(p.Name, dir)
	}

	wg.Wait()
	return outdated
}

// handleOutdated acts on the list of outdated plugins based on the configured update mode.
func handleOutdated(runner *tmux.RealRunner, cfg *config.Config, outdated []string) int {
	switch cfg.UpdateMode {
	case "prompt":
		msg := "TPM: " + strconv.Itoa(len(outdated)) + " plugin update(s) available. Press prefix+U to update."
		_ = runner.DisplayMessage(msg)

	case "auto":
		return autoUpdatePlugins(runner, cfg, outdated)
	}

	return 0
}

// autoUpdatePlugins performs automatic updates for the given outdated plugins.
func autoUpdatePlugins(runner *tmux.RealRunner, cfg *config.Config, outdated []string) int {
	output := newOutput(false, runner)
	mgr := newManagerDeps(cfg.PluginPath, output)

	allPlugins, err := config.GatherPlugins(runner, config.RealFS{}, cfg.TmuxConf, os.Getenv("HOME"))
	if err != nil {
		_ = runner.DisplayMessage("TPM: auto-update failed: " + err.Error())
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	mgr.Update(ctx, allPlugins, outdated)

	if output.HasFailed() {
		_ = runner.DisplayMessage("TPM: auto-update failed for some plugins: " + strings.Join(outdated, ", "))
		return 1
	}
	_ = runner.DisplayMessage("TPM: " + strconv.Itoa(len(outdated)) + " plugin(s) updated successfully.")
	return 0
}
