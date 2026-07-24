// Package git provides interfaces for git operations used by tpack.
package git

import "context"

// CloneOptions configures a git clone operation.
type CloneOptions struct {
	URL    string
	Dir    string
	Branch string
	Depth  int // 0 means no depth limit
	// OnWarning is called for non-fatal issues (e.g. failed submodule init).
	// May be nil. The clone overall is still considered successful.
	OnWarning func(message string)
}

// PullOptions configures a git pull operation.
type PullOptions struct {
	Dir    string
	Branch string // Optional branch to checkout before pulling
	// OnWarning is called for non-fatal issues (e.g. failed submodule update).
	// May be nil. The pull overall is still considered successful.
	OnWarning func(message string)
}

// Cloner clones git repositories.
type Cloner interface {
	Clone(ctx context.Context, opts CloneOptions) error
}

// Puller pulls updates for an existing repository.
type Puller interface {
	Pull(ctx context.Context, opts PullOptions) (string, error)
}

// Validator checks whether a directory is a valid git repository.
type Validator interface {
	IsGitRepo(dir string) bool
}

// Fetcher checks whether a local repository is behind its remote.
type Fetcher interface {
	IsOutdated(ctx context.Context, dir string) (bool, error)
}

// OriginReader reads a repository's configured origin URL.
type OriginReader interface {
	Origin(ctx context.Context, dir string) (string, error)
}

// Commit represents a single git commit.
type Commit struct {
	Hash    string
	Message string
}

// RevParser resolves git refs to commit hashes.
type RevParser interface {
	RevParse(ctx context.Context, dir string) (string, error)
}

// Logger retrieves commit log entries between two refs.
type Logger interface {
	Log(ctx context.Context, dir, fromRef, toRef string) ([]Commit, error)
}
