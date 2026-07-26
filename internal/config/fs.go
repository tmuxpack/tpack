package config

import (
	"os"
	"path/filepath"
)

// FS abstracts filesystem operations for testability.
type FS interface {
	ReadFile(name string) ([]byte, error)
	FileExists(name string) bool
	IsRegularFile(name string) bool
	Glob(pattern string) ([]string, error)
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

func (RealFS) IsRegularFile(name string) bool {
	info, err := os.Stat(name)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	file, err := os.Open(name)
	if err != nil {
		return false
	}
	return file.Close() == nil
}

func (RealFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
