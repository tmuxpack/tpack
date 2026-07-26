package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmuxpack/tpack/internal/tmux"
)

// SourceDocument is one readable tmux configuration file in source order.
type SourceDocument struct {
	Path    string
	Content string
}

// SourceGraph contains every readable configuration document, loaded once.
type SourceGraph struct {
	documents []SourceDocument
	execution []string
}

// Documents returns a copy of the graph's ordered documents.
func (g SourceGraph) Documents() []SourceDocument {
	return append([]SourceDocument(nil), g.documents...)
}

// Paths returns the ordered paths in the graph.
func (g SourceGraph) Paths() []string {
	paths := make([]string, 0, len(g.documents))
	for _, doc := range g.documents {
		paths = append(paths, doc.Path)
	}
	return paths
}

// SourceReadError reports an unreadable required configuration source.
type SourceReadError struct {
	Parent    string
	Directive string
	Target    string
	Err       error
}

func (e *SourceReadError) Error() string {
	return fmt.Sprintf("cannot read required source %s from %s (directive %q): %v", e.Target, e.Parent, e.Directive, e.Err)
}

func (e *SourceReadError) Unwrap() error { return e.Err }

// LoadSourceGraph recursively loads system, user, and sourced tmux configurations.
func LoadSourceGraph(runner tmux.Runner, fs FS, paths Paths) (SourceGraph, error) {
	var graph SourceGraph
	type documentState struct {
		documentAt int
		directives []locatedSourceDirective
		executed   bool
	}
	loaded := make(map[string]*documentState)

	var visit func(path, parent, directive string, optional, execute bool) error
	executeDocument := func(path string, state *documentState) error {
		state.executed = true
		content := graph.documents[state.documentAt].Content
		cursor := 0
		for _, source := range state.directives {
			graph.execution = append(graph.execution, content[cursor:source.start])
			if err := visitSourceDirectives(runner, fs, paths, path, []locatedSourceDirective{source}, visit); err != nil {
				return err
			}
			cursor = source.end
		}
		graph.execution = append(graph.execution, content[cursor:])
		return nil
	}
	visit = func(path, parent, directive string, optional, execute bool) error {
		clean := filepath.Clean(path)
		if state, ok := loaded[clean]; ok {
			if !execute || state.executed {
				return nil
			}
			return executeDocument(clean, state)
		}
		data, err := fs.ReadFile(clean)
		if err != nil {
			if optional {
				return nil
			}
			return &SourceReadError{Parent: parent, Directive: directive, Target: clean, Err: err}
		}

		content := string(data)
		directives, err := parseLocatedSourceDirectives(clean, content, runner.ExpandFormat)
		if err != nil {
			return err
		}
		state := &documentState{
			documentAt: len(graph.documents),
			directives: directives,
			executed:   execute,
		}
		loaded[clean] = state
		graph.documents = append(graph.documents, SourceDocument{Path: clean, Content: content})
		if !execute {
			return nil
		}
		return executeDocument(clean, state)
	}

	roots := tmuxConfigRoots(paths)
	if !containsPath(roots, "/etc/tmux.conf") {
		if err := visit("/etc/tmux.conf", "", "/etc/tmux.conf", true, true); err != nil {
			return SourceGraph{}, err
		}
	}
	for _, root := range roots {
		if err := visit(root, root, root, false, true); err != nil {
			return SourceGraph{}, err
		}
	}
	return graph, nil
}

func visitSourceDirectives(
	runner tmux.Runner,
	fs FS,
	paths Paths,
	parent string,
	directives []locatedSourceDirective,
	visit func(path, parent, directive string, optional, execute bool) error,
) error {
	for _, source := range directives {
		for _, sourcePath := range source.Paths {
			target, err := expandSourcePath(runner, sourcePath, source.ExpandFormats, paths)
			if err != nil {
				return configSyntaxError(parent, source.Line, "expand source path: "+err.Error())
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(parent), target)
			}
			matches, err := sourceMatches(fs, target, source.Optional)
			if err != nil {
				if source.Optional {
					return configSyntaxError(parent, source.Line, "expand source glob: "+err.Error())
				}
				return &SourceReadError{Parent: parent, Directive: source.Text, Target: target, Err: err}
			}
			for _, match := range matches {
				if err := visit(match, parent, source.Text, source.Optional, !source.ParseOnly); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func expandSourcePath(runner tmux.Runner, raw string, useFormats bool, paths Paths) (string, error) {
	value := raw
	if useFormats {
		expanded, err := runner.ExpandFormat(value)
		if err != nil {
			return "", err
		}
		value = expanded
	}

	switch {
	case value == "~":
		value = paths.Home
	case strings.HasPrefix(value, "~/"):
		value = filepath.Join(paths.Home, value[2:])
	}
	value = expandTmuxEnvironment(runner, value)
	return filepath.Clean(value), nil
}

func expandTmuxEnvironment(runner tmux.Runner, value string) string {
	return os.Expand(value, func(name string) string {
		if expanded, err := runner.ShowEnvironment(name); err == nil {
			return expanded
		}
		return os.Getenv(name)
	})
}

func sourceMatches(fs FS, expanded string, optional bool) ([]string, error) {
	matches, err := fs.Glob(expanded)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) == 0 && !optional {
		return []string{expanded}, nil
	}
	return matches, nil
}

func tmuxConfigRoots(paths Paths) []string {
	if len(paths.TmuxConfs) != 0 {
		return paths.TmuxConfs
	}
	if paths.TmuxConf != "" {
		return []string{paths.TmuxConf}
	}
	return nil
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == target {
			return true
		}
	}
	return false
}
