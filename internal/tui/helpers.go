package tui

import (
	"fmt"
	"os"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

// buildPluginItems converts raw plugins into enriched PluginItems with status.
func buildPluginItems(plugins []plug.Plugin, pluginPath plug.Root, validator git.Validator, loadErrors *loadErrorIndex) []PluginItem {
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
			msg, ok := loadErrors.lookup(p)
			if ok {
				item.Status = StatusLoadFailed
				item.LoadErr = msg
			}
		}
		items = append(items, item)
	}
	return items
}

type loadErrorIndex struct {
	byDirName    map[string]string
	byLegacyName map[string]string
}

func (i *loadErrorIndex) lookup(plugin plug.Plugin) (string, bool) {
	if i == nil {
		return "", false
	}
	if msg, ok := i.byDirName[plugin.DirName]; ok {
		return msg, true
	}
	legacyName := plug.LegacyPluginName(plugin.Spec)
	if plugin.Alias != "" {
		legacyName = plugin.Alias
	}
	msg, ok := i.byLegacyName[legacyName]
	return msg, ok
}

// loadErrorMap indexes new load failures by directory name and keeps legacy
// name-only records separate so only those records participate in name fallback.
func loadErrorMap(failures []plug.LoadFailure) *loadErrorIndex {
	if len(failures) == 0 {
		return nil
	}
	index := &loadErrorIndex{
		byDirName:    make(map[string]string, len(failures)),
		byLegacyName: make(map[string]string, len(failures)),
	}
	for _, f := range failures {
		if f.DirName == "" {
			index.byLegacyName[f.Name] = f.Message
			continue
		}
		index.byDirName[f.DirName] = f.Message
	}
	return index
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
		items[i] = OrphanItem{Name: o.Name, DirName: o.Name, Path: o.Path}
	}
	return items, nil
}
