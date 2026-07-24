package plug

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Root is a validated absolute plugin directory.
type Root struct {
	path string
}

// RootError describes why a path cannot be used as a plugin root.
type RootError struct {
	Source string
	Value  string
	Reason string
}

func (e *RootError) Error() string {
	return fmt.Sprintf("invalid plugin path from %s %q: %s", e.Source, e.Value, e.Reason)
}

// NewRoot expands and validates a plugin root.
func NewRoot(source, raw, home, xdgConfigHome string) (Root, error) {
	expanded := strings.TrimSpace(ManualExpansion(raw, home, xdgConfigHome))
	if expanded == "" {
		return Root{}, &RootError{Source: source, Value: raw, Reason: "path is empty"}
	}
	if !filepath.IsAbs(expanded) {
		return Root{}, &RootError{Source: source, Value: raw, Reason: "path must be absolute"}
	}

	clean := filepath.Clean(expanded)
	volumeRoot := filepath.VolumeName(clean) + string(filepath.Separator)
	if clean == volumeRoot {
		return Root{}, &RootError{Source: source, Value: raw, Reason: "filesystem root is not allowed"}
	}
	return Root{path: clean}, nil
}

// Path returns the canonical path without a trailing separator.
func (r Root) Path() (string, error) {
	if r.path == "" {
		return "", &RootError{Source: "zero value", Reason: "path is empty"}
	}
	return r.path, nil
}

// String returns the canonical path with a trailing separator for compatibility.
func (r Root) String() string {
	if r.path == "" {
		return ""
	}
	return r.path + string(filepath.Separator)
}

// Child returns an exact plugin directory component below the root.
func (r Root) Child(name string) (string, error) {
	root, err := r.Path()
	if err != nil {
		return "", err
	}
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) ||
		filepath.VolumeName(name) != "" || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid plugin name %q", name)
	}
	child := filepath.Join(root, name)
	rel, err := filepath.Rel(root, child)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("plugin path %q escapes root %q", child, root)
	}
	return child, nil
}

// Resolved returns a validated root after resolving filesystem symlinks.
func (r Root) Resolved() (Root, error) {
	path, err := r.Path()
	if err != nil {
		return Root{}, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Root{}, fmt.Errorf("resolve plugin root %s: %w", path, err)
	}
	return NewRoot("resolved plugin path", resolved, "", "")
}
