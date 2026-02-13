package git

import "context"

// CloneWithFallback tries cloning with the given options, and on failure
// normalizes the URL using normalize and retries.
func CloneWithFallback(ctx context.Context, cloner Cloner, opts CloneOptions, normalize func(string) string) error {
	err := cloner.Clone(ctx, opts)
	if err == nil {
		return nil
	}
	opts.URL = normalize(opts.URL)
	return cloner.Clone(ctx, opts)
}
