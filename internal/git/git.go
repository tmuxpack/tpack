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
	Dir string
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

// FetchOptions configures a git fetch operation.
type FetchOptions struct {
	Dir string
}

// Fetcher fetches from the remote and checks if the local branch is behind.
type Fetcher interface {
	Fetch(ctx context.Context, opts FetchOptions) error
	IsOutdated(dir string) (bool, error)
}
