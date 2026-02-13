package config

import (
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// fileConfig is the top-level structure of the config file.
type fileConfig struct {
	Colors ColorConfig `yaml:"colors"`
}

// loadFileConfig reads <xdg>/tpm/config.yml and returns color overrides.
// Returns a zero-value ColorConfig on any error (file missing, parse error, etc.).
func loadFileConfig(o *resolveOpts) ColorConfig {
	path := filepath.Join(o.xdgConfigHome(), "tpm", "config.yml")

	data, err := o.fs.ReadFile(path)
	if err != nil {
		return ColorConfig{}
	}

	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return ColorConfig{}
	}

	return fc.Colors
}
