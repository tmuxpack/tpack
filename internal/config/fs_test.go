package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// MockFS provides configurable filesystem responses for testing.
type MockFS struct {
	Files      map[string]string
	NonRegular map[string]bool
}

func NewMockFS() *MockFS {
	return &MockFS{
		Files:      make(map[string]string),
		NonRegular: make(map[string]bool),
	}
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

func (m *MockFS) IsRegularFile(name string) bool {
	_, ok := m.Files[name]
	return ok && !m.NonRegular[name]
}

func (m *MockFS) Glob(pattern string) ([]string, error) {
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, err
	}

	var matches []string
	for name := range m.Files {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func TestRealFSIsRegularFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tmux.conf")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	fs := RealFS{}
	if !fs.IsRegularFile(file) {
		t.Errorf("IsRegularFile(%q) = false, want true", file)
	}
	if fs.IsRegularFile(dir) {
		t.Errorf("IsRegularFile(%q) = true for a directory", dir)
	}
	if fs.IsRegularFile(filepath.Join(dir, "missing")) {
		t.Error("IsRegularFile returned true for a missing path")
	}
}

func TestRealFSIsRegularFileRejectsUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read permissionless files")
	}
	file := filepath.Join(t.TempDir(), "tmux.conf")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o000); err != nil {
		t.Fatal(err)
	}
	if (RealFS{}).IsRegularFile(file) {
		t.Errorf("IsRegularFile(%q) = true for unreadable file", file)
	}
}

func TestMockFSGlobReturnsSortedMatches(t *testing.T) {
	fs := NewMockFS()
	fs.Files["/config/20-second.conf"] = ""
	fs.Files["/config/10-first.conf"] = ""
	fs.Files["/config/ignored.txt"] = ""

	got, err := fs.Glob("/config/*.conf")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/config/10-first.conf", "/config/20-second.conf"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Glob() = %v, want %v", got, want)
	}
}

func TestMockFSGlobRejectsMalformedPattern(t *testing.T) {
	fs := NewMockFS()
	if _, err := fs.Glob("["); !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("Glob() error = %v, want ErrBadPattern", err)
	}
}
