package tui

import (
	"fmt"
	"os"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

// buildPluginItems converts raw plugins into enriched PluginItems with status.
// loadErrors maps plugin name → load-error message; an installed plugin with an
// entry is marked StatusLoadFailed.
func buildPluginItems(plugins []plug.Plugin, pluginPath plug.Root, validator git.Validator, loadErrors map[string]string) []PluginItem {
	items := make([]PluginItem, 0, len(plugins))
	for _, p := range plugins {
		status := StatusNotInstalled
		dir, pathErr := pluginPath.Child(p.Name)
		if pathErr != nil {
			items = append(items, PluginItem{Name: p.Name, Spec: p.Spec, Branch: p.Branch, Status: StatusLoadFailed, LoadErr: pathErr.Error()})
			continue
		}
		info, err := os.Stat(dir)
		installed := err == nil && info.IsDir() && validator.IsGitRepo(dir)
		if installed {
			status = StatusChecking
		}
		item := PluginItem{
			Name:   p.Name,
			Spec:   p.Spec,
			Branch: p.Branch,
			Status: status,
		}
		if installed {
			if msg, ok := loadErrors[p.Name]; ok {
				item.Status = StatusLoadFailed
				item.LoadErr = msg
			}
		}
		items = append(items, item)
	}
	return items
}

// loadErrorMap indexes load failures by plugin name.
func loadErrorMap(failures []plug.LoadFailure) map[string]string {
	if len(failures) == 0 {
		return nil
	}
	m := make(map[string]string, len(failures))
	for _, f := range failures {
		m[f.Name] = f.Message
	}
	return m
}

// findOrphans resolves the plugin root before returning orphan items for the TUI.
func findOrphans(plugins []plug.Plugin, pluginPath plug.Root) ([]OrphanItem, error) {
	resolved, err := pluginPath.Resolved()
	if err != nil {
		return nil, fmt.Errorf("unsafe plugin directory: %w", err)
	}
	shared, err := plug.FindOrphans(plugins, resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect plugin directory: %w", err)
	}
	items := make([]OrphanItem, len(shared))
	for i, o := range shared {
		items[i] = OrphanItem{Name: o.Name, Path: o.Path}
	}
	return items, nil
}
