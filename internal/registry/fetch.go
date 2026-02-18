package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const cacheFile = "registry.yml"

// DefaultRegistryURL is the URL to the CI-generated merged registry file.
const DefaultRegistryURL = "https://raw.githubusercontent.com/tmuxpack/plugins-registry/main/dist/plugins.yml"

// DefaultCacheTTL is the default time-to-live for the local registry cache.
const DefaultCacheTTL = 6 * time.Hour

// Fetch retrieves the registry from the remote URL and caches it locally.
// If the cache is fresh (within ttl), it returns the cached version.
// If the remote fetch fails, it falls back to any existing cache.
// A zero ttl forces a remote fetch.
func Fetch(ctx context.Context, url string, cacheDir string, ttl time.Duration) (*Registry, error) {
	cachePath := filepath.Join(cacheDir, cacheFile)

	// Check cache freshness.
	if ttl > 0 {
		if info, err := os.Stat(cachePath); err == nil {
			if time.Since(info.ModTime()) < ttl {
				data, err := os.ReadFile(cachePath)
				if err == nil {
					return Parse(data)
				}
			}
		}
	}

	// Fetch from remote.
	data, err := fetchRemote(ctx, url)
	if err != nil {
		// Fall back to cache.
		cached, cacheErr := os.ReadFile(cachePath)
		if cacheErr != nil {
			return nil, fmt.Errorf("fetch registry: %w (no cache available)", err)
		}
		return Parse(cached)
	}

	// Validate before caching.
	reg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	// Write cache.
	_ = os.MkdirAll(cacheDir, 0o755)
	_ = os.WriteFile(cachePath, data, 0o600)

	return reg, nil
}

func fetchRemote(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
