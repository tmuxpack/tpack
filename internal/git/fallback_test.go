package git_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
)

// recordingCloner records every CloneOptions it receives and returns errors
// from a per-call slice, allowing fine-grained control over each attempt.
type recordingCloner struct {
	calls []git.CloneOptions
	errs  []error // errs[i] is returned on the i-th call; nil means success
}

func (r *recordingCloner) Clone(_ context.Context, opts git.CloneOptions) error {
	idx := len(r.calls)
	r.calls = append(r.calls, opts)
	if idx < len(r.errs) {
		return r.errs[idx]
	}
	return nil
}

func TestCloneWithFallback(t *testing.T) {
	const (
		originalURL   = "https://example.com/repo.git"
		normalizedURL = "git@example.com:repo.git"
		dir           = "/tmp/test"
	)

	errClone := errors.New("clone failed")

	tests := []struct {
		name           string
		errs           []error
		wantErr        bool
		wantCalls      int
		wantNormalized bool
		wantSecondURL  string
	}{
		{
			name:           "first clone succeeds without fallback",
			errs:           []error{nil},
			wantErr:        false,
			wantCalls:      1,
			wantNormalized: false,
		},
		{
			name:           "first clone fails and fallback succeeds with normalized URL",
			errs:           []error{errClone, nil},
			wantErr:        false,
			wantCalls:      2,
			wantNormalized: true,
			wantSecondURL:  normalizedURL,
		},
		{
			name:           "both clones fail and error is returned",
			errs:           []error{errClone, errClone},
			wantErr:        true,
			wantCalls:      2,
			wantNormalized: true,
			wantSecondURL:  normalizedURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloner := &recordingCloner{errs: tt.errs}

			normalizeCalled := false
			normalize := func(url string) string {
				normalizeCalled = true
				if url != originalURL {
					t.Errorf("normalize received URL %q, want %q", url, originalURL)
				}
				return normalizedURL
			}

			err := git.CloneWithFallback(context.Background(), cloner, git.CloneOptions{
				URL: originalURL,
				Dir: dir,
			}, normalize)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CloneWithFallback() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(cloner.calls) != tt.wantCalls {
				t.Fatalf("expected %d clone calls, got %d", tt.wantCalls, len(cloner.calls))
			}

			// First call should always use the original URL.
			if cloner.calls[0].URL != originalURL {
				t.Errorf("first call URL = %q, want %q", cloner.calls[0].URL, originalURL)
			}

			if normalizeCalled != tt.wantNormalized {
				t.Errorf("normalize called = %v, want %v", normalizeCalled, tt.wantNormalized)
			}

			// When fallback occurs, verify the second call uses the normalized URL.
			if tt.wantCalls == 2 {
				if cloner.calls[1].URL != tt.wantSecondURL {
					t.Errorf("second call URL = %q, want %q", cloner.calls[1].URL, tt.wantSecondURL)
				}
			}
		})
	}
}
