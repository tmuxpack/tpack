package registry

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Registry holds the full plugin registry fetched from the remote source.
type Registry struct {
	Categories []string       `yaml:"categories"`
	Plugins    []RegistryItem `yaml:"plugins"`
}

// RegistryItem represents a single plugin in the registry.
type RegistryItem struct {
	Repo        string `yaml:"repo"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	Category    string `yaml:"category"`
	Stars       int    `yaml:"stars"`
}

// Parse deserializes raw YAML bytes into a Registry.
func Parse(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// Search returns plugins matching the query against repo name and description.
// Results are sorted by stars descending. An empty query returns all plugins.
func Search(reg *Registry, query string) []RegistryItem {
	if query == "" {
		results := make([]RegistryItem, len(reg.Plugins))
		copy(results, reg.Plugins)
		sortByStars(results)
		return results
	}

	q := strings.ToLower(query)
	var results []RegistryItem
	for _, p := range reg.Plugins {
		if strings.Contains(strings.ToLower(p.Repo), q) ||
			strings.Contains(strings.ToLower(p.Description), q) {
			results = append(results, p)
		}
	}
	sortByStars(results)
	return results
}

// FilterByCategory returns plugins belonging to the given category.
func FilterByCategory(reg *Registry, category string) []RegistryItem {
	var results []RegistryItem
	for _, p := range reg.Plugins {
		if p.Category == category {
			results = append(results, p)
		}
	}
	return results
}

func sortByStars(items []RegistryItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Stars > items[j].Stars
	})
}
