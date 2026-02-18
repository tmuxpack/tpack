package config

import "os"

// FS abstracts filesystem operations for testability.
type FS interface {
	ReadFile(name string) ([]byte, error)
	FileExists(name string) bool
}

// RealFS implements FS using the real filesystem.
type RealFS struct{}

func (RealFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (RealFS) FileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
