// Package git provides interfaces for git operations used by TPM.
package git

import "context"

// CloneOptions configures a git clone operation.
type CloneOptions struct {
	URL    string
	Dir    string
	Branch string
}

// PullOptions configures a git pull operation.
type PullOptions struct {
	Dir    string
	Branch string // Optional branch to checkout before pulling
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
