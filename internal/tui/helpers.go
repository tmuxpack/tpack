package tui

import (
	"fmt"
	"os"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

// buildPluginItems converts raw plugins into enriched PluginItems with status.
// loadErrors maps plugin directory name → load-error message; an installed plugin with an
// entry is marked StatusLoadFailed.
func buildPluginItems(plugins []plug.Plugin, pluginPath plug.Root, validator git.Validator, loadErrors map[string]string) []PluginItem {
	items := make([]PluginItem, 0, len(plugins))
	for _, p := range plugins {
		status := StatusNotInstalled
		dir, pathErr := pluginPath.Child(p.DirName)
		if pathErr != nil {
			items = append(items, PluginItem{
				Raw: p.Raw, Name: p.Name, Identity: p.Identity, DirName: p.DirName,
				Spec: p.Spec, Branch: p.Branch, Status: StatusLoadFailed, LoadErr: pathErr.Error(),
			})
			continue
		}
		info, err := os.Stat(dir)
		installed := err == nil && info.IsDir() && validator.IsGitRepo(dir)
		if installed {
			status = StatusChecking
		}
		item := PluginItem{
			Raw:      p.Raw,
			Name:     p.Name,
			Identity: p.Identity,
			DirName:  p.DirName,
			Spec:     p.Spec,
			Branch:   p.Branch,
			Status:   status,
		}
		if installed {
			msg, ok := loadErrors[p.DirName]
			if !ok {
				msg, ok = loadErrors[p.Name]
			}
			if ok {
				item.Status = StatusLoadFailed
				item.LoadErr = msg
			}
		}
		items = append(items, item)
	}
	return items
}

// loadErrorMap indexes load failures by directory name, falling back to the
// legacy display name for records written before directory keys were stored.
func loadErrorMap(failures []plug.LoadFailure) map[string]string {
	if len(failures) == 0 {
		return nil
	}
	m := make(map[string]string, len(failures))
	for _, f := range failures {
		key := f.DirName
		if key == "" {
			key = f.Name
		}
		m[key] = f.Message
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
