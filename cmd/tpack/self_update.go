package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/state"
	"github.com/tmuxpack/tpack/internal/tmux"
)

const (
	selfUpdateInterval = 24 * time.Hour
	selfUpdateTimeout  = 30 * time.Second
	githubAPIURL       = "https://api.github.com/repos/tmuxpack/tpack/releases/latest"
	githubDownloadURL  = "https://github.com/tmuxpack/tpack/releases/download"
	maxBinarySize      = 50 * 1024 * 1024 // 50 MiB safety limit for extracted binary
)

// selfUpdateResult represents the outcome of a self-update check.
type selfUpdateResult int

const (
	selfUpdateSkipped selfUpdateResult = iota
	selfUpdateSuccess
	selfUpdateFailed
)

// selfUpdateParams holds parameters for selfUpdateCheck, enabling testability.
type selfUpdateParams struct {
	statePath   string
	version     string // current version (from main.version)
	binaryPath  string // path to the current binary
	apiURL      string // GitHub API URL (overridable for tests)
	downloadURL string // download URL template (overridable for tests)
	repoDir     string // tpack repo directory for git sync
	skipGitSync bool   // skip git checkout (for tests)
}

// githubRelease represents the relevant fields from the GitHub releases API.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// runSelfUpdate is the entry point for the `tpack self-update` command.
func runSelfUpdate() int {
	runner := tmux.NewRealRunner()

	cfg, err := config.Resolve(runner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tpack: config error:", err)
		return 1
	}

	binary := findBinary()
	repoDir := filepath.Dir(binary) // tpack repo is the directory containing the binary

	p := selfUpdateParams{
		statePath:   cfg.StatePath,
		version:     version,
		binaryPath:  binary,
		apiURL:      githubAPIURL,
		downloadURL: githubDownloadURL,
		repoDir:     repoDir,
	}

	result := selfUpdateCheck(p, runner)

	switch result {
	case selfUpdateSuccess, selfUpdateSkipped:
		return 0
	case selfUpdateFailed:
		return 1
	}

	return 1
}

// selfUpdateCheck orchestrates the self-update flow.
func selfUpdateCheck(p selfUpdateParams, runner tmux.Runner) selfUpdateResult {
	// 1. Load state, check LastSelfUpdateCheck -- if <24h ago, skip.
	st := state.Load(p.statePath)
	if !st.LastSelfUpdateCheck.IsZero() && time.Since(st.LastSelfUpdateCheck) < selfUpdateInterval {
		return selfUpdateSkipped
	}

	// 2. Save timestamp immediately to prevent retry storms.
	st.LastSelfUpdateCheck = time.Now()
	_ = state.Save(p.statePath, st)

	// 3. Fetch latest release version from GitHub API.
	latest, err := fetchLatestVersion(p.apiURL)
	if err != nil {
		_ = runner.DisplayMessage("tpack: self-update failed (download error)")
		return selfUpdateFailed
	}

	// 4. Compare against current version -- if same or dev build, skip.
	currentVersion := strings.TrimPrefix(p.version, "v")
	if currentVersion == latest || currentVersion == "dev" {
		return selfUpdateSkipped
	}

	// 5-6. Download and extract the new binary.
	archiveURL := fmt.Sprintf("%s/v%s/tpack_%s_%s_%s.tar.gz",
		p.downloadURL, latest, latest, runtime.GOOS, runtime.GOARCH)

	newBinaryPath, cleanup, err := downloadAndExtract(archiveURL)
	if err != nil {
		_ = runner.DisplayMessage("tpack: self-update failed (extract error)")
		return selfUpdateFailed
	}
	defer cleanup()

	// 7. Atomic replace: rename temp binary over current binary.
	if err := os.Rename(newBinaryPath, p.binaryPath); err != nil {
		_ = runner.DisplayMessage("tpack: self-update failed (permission error)")
		return selfUpdateFailed
	}

	// 8. Git checkout matching tag to sync shell scripts.
	tag := "v" + latest
	if !p.skipGitSync {
		if err := syncGitRepo(p.repoDir, tag); err != nil {
			_ = runner.DisplayMessage(fmt.Sprintf("tpack: updated to %s (warning: repo sync failed)", tag))
			return selfUpdateSuccess
		}
	}

	// 9. Display success message.
	_ = runner.DisplayMessage(fmt.Sprintf("tpack: updated to %s", tag))
	return selfUpdateSuccess
}

// fetchLatestVersion calls the GitHub API and returns the latest version
// without the "v" prefix.
func fetchLatestVersion(apiURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// downloadAndExtract downloads a tar.gz archive and extracts the Go binary
// to a temp directory. Returns the path to the extracted binary,
// a cleanup function, and any error.
func downloadAndExtract(url string) (string, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("downloading archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "tpack-update-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	// Cap download to maxBinarySize + overhead for archive compression and headers.
	limitedBody := io.LimitReader(resp.Body, maxBinarySize+1024*1024)

	binaryPath, err := extractBinaryFromArchive(limitedBody, tmpDir)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}

// extractBinaryFromArchive reads a gzip+tar stream and extracts the Go binary
// entry to the given directory.
func extractBinaryFromArchive(r io.Reader, destDir string) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar: %w", err)
		}

		// Only extract regular files named tpack (skip symlinks, dirs, etc.).
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != binaryName {
			continue
		}

		if hdr.Size > maxBinarySize {
			return "", fmt.Errorf("binary too large: %d bytes (max %d)", hdr.Size, maxBinarySize)
		}

		dest := filepath.Join(destDir, binaryName)

		f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // executable binary needs 0755
		if err != nil {
			return "", fmt.Errorf("creating file: %w", err)
		}

		// Limit copy to declared header size for safety.
		if _, err := io.Copy(f, io.LimitReader(tr, hdr.Size)); err != nil {
			_ = f.Close()
			return "", fmt.Errorf("extracting file: %w", err)
		}

		if err := f.Close(); err != nil {
			return "", fmt.Errorf("closing file: %w", err)
		}

		return dest, nil
	}

	return "", fmt.Errorf("%s not found in archive", binaryName)
}

// syncGitRepo runs `git checkout <tag>` in the given repository directory.
func syncGitRepo(repoDir, tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", tag) //nolint:gosec // tag comes from GitHub API
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
