package config

import (
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// UpdateConfig holds update-check settings from the config file.
type UpdateConfig struct {
	CheckInterval string `yaml:"check_interval"`
	Mode          string `yaml:"mode"`
}

// fileConfig is the top-level structure of the config file.
type fileConfig struct {
	Colors  ColorConfig  `yaml:"colors"`
	Updates UpdateConfig `yaml:"updates"`
}

// loadFileConfig reads <xdg>/tpm/config.yml and returns the parsed config.
// Returns a zero-value fileConfig on any error (file missing, parse error, etc.).
func loadFileConfig(o *resolveOpts) fileConfig {
	path := filepath.Join(o.xdgConfigHome(), "tpm", "config.yml")

	data, err := o.fs.ReadFile(path)
	if err != nil {
		return fileConfig{}
	}

	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fileConfig{}
	}

	return fc
}

// parseCheckInterval parses a duration string, returning 0 on any error.
func parseCheckInterval(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0
	}
	return d
}
