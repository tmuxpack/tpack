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
	Host        string `yaml:"host,omitempty"`
	AddedDate   string `yaml:"added_date,omitempty"`
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

// Newest returns the n most recently added plugins, sorted by added_date
// descending, then by stars descending within the same date. Entries with
// empty added_date sort last.
func Newest(reg *Registry, n int) []RegistryItem {
	items := make([]RegistryItem, len(reg.Plugins))
	copy(items, reg.Plugins)

	sort.SliceStable(items, func(i, j int) bool {
		di, dj := items[i].AddedDate, items[j].AddedDate
		switch {
		case di == "" && dj == "":
			return items[i].Stars > items[j].Stars
		case di == "":
			return false
		case dj == "":
			return true
		case di != dj:
			return di > dj
		default:
			return items[i].Stars > items[j].Stars
		}
	})

	if n > len(items) {
		n = len(items)
	}
	return items[:n]
}

// ExcludeCategories returns a new Registry with the given categories and their
// plugins removed.
func ExcludeCategories(reg *Registry, exclude []string) *Registry {
	if len(exclude) == 0 {
		return reg
	}
	hidden := make(map[string]bool, len(exclude))
	for _, c := range exclude {
		hidden[c] = true
	}
	var cats []string
	for _, c := range reg.Categories {
		if !hidden[c] {
			cats = append(cats, c)
		}
	}
	var plugins []RegistryItem
	for _, p := range reg.Plugins {
		if !hidden[p.Category] {
			plugins = append(plugins, p)
		}
	}
	return &Registry{Categories: cats, Plugins: plugins}
}

func sortByStars(items []RegistryItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Stars > items[j].Stars
	})
}
