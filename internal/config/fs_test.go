package config

import "os"

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
