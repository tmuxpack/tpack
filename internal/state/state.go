// Package state manages TPM persistent state (e.g. last update check timestamp).
package state

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const stateFile = "state.yml"

// State holds persistent TPM state.
type State struct {
	LastUpdateCheck     time.Time `yaml:"last_update_check"`
	LastSelfUpdateCheck time.Time `yaml:"last_self_update_check"`
}

// Load reads state from statePath/state.yml.
// Returns zero-value State on any error.
func Load(statePath string) State {
	data, err := os.ReadFile(filepath.Join(statePath, stateFile))
	if err != nil {
		return State{}
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return State{}
	}
	return s
}

// Save writes state to statePath/state.yml, creating directories as needed.
func Save(statePath string, s State) error {
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(statePath, stateFile), data, 0o600)
}
