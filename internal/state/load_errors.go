package state

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tmuxpack/tpack/internal/plug"
)

const loadErrorsFile = "load-errors.yml"

// loadErrorsDoc is the on-disk schema for load-errors.yml.
type loadErrorsDoc struct {
	LoadErrors []plug.LoadFailure `yaml:"load_errors"`
}

// SaveLoadErrors writes plugin load failures to statePath/load-errors.yml,
// replacing any previous content. Empty failures removes the file; an empty
// statePath is a no-op.
func SaveLoadErrors(statePath string, failures []plug.LoadFailure) error {
	if statePath == "" {
		return nil
	}
	p := filepath.Join(statePath, loadErrorsFile)
	if len(failures) == 0 {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(loadErrorsDoc{LoadErrors: failures})
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// LoadLoadErrors reads plugin load failures from statePath/load-errors.yml.
// Returns nil on any error (missing or corrupt file). warn, if non-nil,
// receives a message when the file exists but cannot be parsed.
func LoadLoadErrors(statePath string, warn func(string)) []plug.LoadFailure {
	if statePath == "" {
		return nil
	}
	p := filepath.Join(statePath, loadErrorsFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var doc loadErrorsDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		if warn != nil {
			warn(fmt.Sprintf("corrupt load-errors file %s: %v", p, err))
		}
		return nil
	}
	return doc.LoadErrors
}
