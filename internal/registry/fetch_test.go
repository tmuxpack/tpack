package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testRegistryYAML = `categories:
  - theme
plugins:
  - repo: catppuccin/tmux
    description: Soothing pastel theme
    author: catppuccin
    category: theme
    stars: 1250
`

func TestFetch_FromServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRegistryYAML))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	reg, err := Fetch(context.Background(), srv.URL, cacheDir, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(reg.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(reg.Plugins))
	}

	// Verify cache was written.
	cached, err := os.ReadFile(filepath.Join(cacheDir, cacheFile))
	if err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
	if len(cached) == 0 {
		t.Error("cache file is empty")
	}
}

func TestFetch_FromCache(t *testing.T) {
	cacheDir := t.TempDir()
	// Pre-populate cache.
	if err := os.WriteFile(filepath.Join(cacheDir, cacheFile), []byte(testRegistryYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	// Server returns error -- should fall back to cache.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	reg, err := Fetch(context.Background(), srv.URL, cacheDir, 0)
	if err != nil {
		t.Fatalf("Fetch with cache fallback: %v", err)
	}
	if len(reg.Plugins) != 1 {
		t.Errorf("expected 1 plugin from cache, got %d", len(reg.Plugins))
	}
}

func TestFetch_CacheTTL(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(testRegistryYAML))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()

	// First fetch -- hits server.
	_, err := Fetch(context.Background(), srv.URL, cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 server call, got %d", callCount)
	}

	// Second fetch -- should use cache (TTL not expired).
	_, err = Fetch(context.Background(), srv.URL, cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 {
		t.Errorf("expected no additional server call, got %d total", callCount)
	}
}

func TestFetch_NoCacheNoServer(t *testing.T) {
	cacheDir := t.TempDir()
	// No server, no cache -- should error.
	_, err := Fetch(context.Background(), "http://127.0.0.1:1", cacheDir, 0)
	if err == nil {
		t.Fatal("expected error with no cache and unreachable server")
	}
}
