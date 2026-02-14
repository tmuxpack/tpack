package plugin

import (
	"os"
	"path/filepath"
)

// Orphan represents a plugin directory not listed in the config.
type Orphan struct {
	Name string
	Path string
}

// FindOrphans returns directories in pluginPath that don't match any plugin name.
func FindOrphans(plugins []Plugin, pluginPath string) []Orphan {
	nameSet := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		nameSet[p.Name] = true
	}

	entries, err := os.ReadDir(pluginPath)
	if err != nil {
		return nil
	}

	var orphans []Orphan
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := PluginName(entry.Name())
		if name == "tpm" || nameSet[name] {
			continue
		}
		orphans = append(orphans, Orphan{
			Name: name,
			Path: filepath.Join(pluginPath, entry.Name()),
		})
	}
	return orphans
}
