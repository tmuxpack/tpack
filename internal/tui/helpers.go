package tui

import (
	"os"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plug"
)

// buildPluginItems converts raw plugins into enriched PluginItems with status.
func buildPluginItems(plugins []plug.Plugin, pluginPath string, validator git.Validator) []PluginItem {
	items := make([]PluginItem, 0, len(plugins))
	for _, p := range plugins {
		status := StatusNotInstalled
		dir := plug.PluginPath(p.Name, pluginPath)
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() && validator.IsGitRepo(dir) {
			status = StatusChecking
		}
		items = append(items, PluginItem{
			Name:   p.Name,
			Spec:   p.Spec,
			Branch: p.Branch,
			Status: status,
		})
	}
	return items
}

// findOrphans returns orphan items for the TUI.
func findOrphans(plugins []plug.Plugin, pluginPath string) []OrphanItem {
	shared := plug.FindOrphans(plugins, pluginPath)
	items := make([]OrphanItem, len(shared))
	for i, o := range shared {
		items[i] = OrphanItem{Name: o.Name, Path: o.Path}
	}
	return items
}
