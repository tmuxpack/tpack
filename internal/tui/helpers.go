package tui

import (
	"os"
	"path/filepath"

	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

// buildPluginItems converts raw plugins into enriched PluginItems with status.
func buildPluginItems(plugins []plugin.Plugin, pluginPath string, validator git.Validator) []PluginItem {
	items := make([]PluginItem, 0, len(plugins))
	for _, p := range plugins {
		status := StatusNotInstalled
		dir := plugin.PluginPath(p.Name, pluginPath)
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() && validator.IsGitRepo(dir) {
			status = StatusInstalled
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

// findOrphans returns directories in pluginPath that are not in the plugin list.
func findOrphans(plugins []plugin.Plugin, pluginPath string) []OrphanItem {
	nameSet := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		nameSet[p.Name] = true
	}

	entries, err := os.ReadDir(pluginPath)
	if err != nil {
		return nil
	}

	var orphans []OrphanItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := plugin.PluginName(entry.Name())
		if name == "tpm" {
			continue
		}
		if nameSet[name] {
			continue
		}
		orphans = append(orphans, OrphanItem{
			Name: name,
			Path: filepath.Join(pluginPath, entry.Name()),
		})
	}
	return orphans
}
