package plug

import "os"

// Represents a plugin directory not listed in the config.
type Orphan struct {
	Name string
	Path string
}

// FindOrphans returns directories in root that don't match any plugin name.
func FindOrphans(plugins []Plugin, root Root) ([]Orphan, error) {
	pluginPath, err := root.Path()
	if err != nil {
		return nil, err
	}
	nameSet := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		nameSet[p.Name] = true
	}

	entries, err := os.ReadDir(pluginPath)
	if err != nil {
		return nil, err
	}

	var orphans []Orphan
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := PluginName(entry.Name())
		if name == "tpm" || name == "tpack" || nameSet[name] {
			continue
		}
		path, err := root.Child(entry.Name())
		if err != nil {
			return nil, err
		}
		orphans = append(orphans, Orphan{Name: name, Path: path})
	}
	return orphans, nil
}
