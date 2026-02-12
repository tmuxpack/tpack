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

// MockFS provides configurable filesystem responses for testing.
type MockFS struct {
	Files map[string]string
}

func NewMockFS() *MockFS {
	return &MockFS{Files: make(map[string]string)}
}

func (m *MockFS) ReadFile(name string) ([]byte, error) {
	content, ok := m.Files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

func (m *MockFS) FileExists(name string) bool {
	_, ok := m.Files[name]
	return ok
}
