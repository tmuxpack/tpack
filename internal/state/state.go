// Package state manages tpack persistent state (e.g. last update check timestamp).
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const stateFile = "state.yml"
const lockFile = "state.lock"

// State holds persistent tpack state.
type State struct {
	LastUpdateCheck     time.Time `yaml:"last_update_check"`
	LastSelfUpdateCheck time.Time `yaml:"last_self_update_check"`
}

// Load reads state from statePath/state.yml.
// Returns zero-value State on any error.
func Load(statePath string) State {
	p := filepath.Join(statePath, stateFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return State{}
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		fmt.Fprintf(os.Stderr, "tpack: warning: corrupt state file %s: %v\n", p, err)
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

// LoadAndSave atomically loads state, applies fn, and saves. An advisory
// file lock serializes concurrent access from background processes.
func LoadAndSave(statePath string, fn func(*State)) error {
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		return err
	}

	lockPath := filepath.Join(statePath, lockFile)
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) //nolint:gosec // file descriptors fit in int on supported platforms
		_ = lf.Close()
	}()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil { //nolint:gosec // file descriptors fit in int on supported platforms
		return fmt.Errorf("acquire lock: %w", err)
	}

	s := Load(statePath)
	fn(&s)
	return Save(statePath, s)
}
