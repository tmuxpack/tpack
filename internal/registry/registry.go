package registry

import "gopkg.in/yaml.v3"

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
