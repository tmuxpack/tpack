package config

import (
	"fmt"
	"path/filepath"

	"github.com/tmuxpack/tpack/internal/plug"
)

// SourceDocument is one readable tmux configuration file in source order.
type SourceDocument struct {
	Path    string
	Content string
}

// SourceGraph contains every readable configuration document, loaded once.
type SourceGraph struct {
	documents []SourceDocument
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
	return fmt.Sprintf("cannot read required source %s from %s: %v", e.Target, e.Parent, e.Err)
}

func (e *SourceReadError) Unwrap() error { return e.Err }

// LoadSourceGraph recursively loads system, user, and sourced tmux configurations.
func LoadSourceGraph(fs FS, paths Paths) (SourceGraph, error) {
	var graph SourceGraph
	visited := make(map[string]bool)

	var visit func(path, parent, directive string, optional bool) error
	visit = func(path, parent, directive string, optional bool) error {
		clean := filepath.Clean(path)
		if visited[clean] {
			return nil
		}
		data, err := fs.ReadFile(clean)
		if err != nil {
			if optional {
				return nil
			}
			return &SourceReadError{Parent: parent, Directive: directive, Target: clean, Err: err}
		}

		visited[clean] = true
		content := string(data)
		graph.documents = append(graph.documents, SourceDocument{Path: clean, Content: content})
		for _, source := range plug.ExtractSourceDirectives(content) {
			target := plug.ManualExpansion(source.Path, paths.Home, paths.XDGConfigHome)
			if err := visit(target, clean, source.Path, source.Optional); err != nil {
				return err
			}
		}
		return nil
	}

	if err := visit("/etc/tmux.conf", "", "/etc/tmux.conf", true); err != nil {
		return SourceGraph{}, err
	}
	if err := visit(paths.TmuxConf, paths.TmuxConf, paths.TmuxConf, false); err != nil {
		return SourceGraph{}, err
	}
	return graph, nil
}
