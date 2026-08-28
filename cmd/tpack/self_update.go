package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/spf13/cobra"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/state"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the tpack binary to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSelfUpdate()
	},
}

const (
	selfUpdateInterval = 24 * time.Hour
	selfUpdateTimeout  = 30 * time.Second
	githubAPIURL       = "https://api.github.com/repos/tmuxpack/tpack/releases/latest"
	githubDownloadURL  = "https://github.com/tmuxpack/tpack/releases/download"
	maxBinarySize      = 50 * 1024 * 1024 // 50 MiB safety limit for extracted binary
)

// Represents the outcome of a self-update check.
type selfUpdateResult int

const (
	selfUpdateSkipped selfUpdateResult = iota
	selfUpdateSuccess
	selfUpdateFailed
)

// Holds parameters for selfUpdateCheck, enabling testability.
type selfUpdateParams struct {
	statePath   string
	version     string // current version (from main.version)
	binaryPath  string // path to the current binary
	apiURL      string // GitHub API URL (overridable for tests)
	downloadURL string // download URL template (overridable for tests)
	repoDir     string // tpack repo directory for git sync
	skipGitSync bool   // skip git checkout (for tests)
}

// Represents the relevant fields from the GitHub releases API.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// Entry point for the `tpack self-update` command.
func runSelfUpdate() error {
	runner := tmux.NewRealRunner()
	output := ui.NewStatusOutput(runner)

	cfg, err := config.Resolve(runner)
	if err != nil {
		output.Err("config: " + err.Error())
		return selfUpdateCommandResult(selfUpdateFailed, output)
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

	result := selfUpdateCheck(p, output)
	return selfUpdateCommandResult(result, output)
}

func selfUpdateCommandResult(result selfUpdateResult, output ui.Output) error {
	if err := outputResult(output); err != nil {
		return err
	}
	switch result {
	case selfUpdateSuccess, selfUpdateSkipped:
		return nil
	case selfUpdateFailed:
		return errSilent
	}

	return errSilent
}

func formatSelfUpdateReplaceError(err error) string {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		err = linkErr.Err
	}

	return fmt.Sprintf("self-update failed (replace binary: %v)", err)
}

// Orchestrates the self-update flow.
func selfUpdateCheck(p selfUpdateParams, output ui.Output) selfUpdateResult {
	// 1. Load state, check LastSelfUpdateCheck -- if <24h ago, skip.
	st := state.Load(p.statePath, nil)
	if !st.LastSelfUpdateCheck.IsZero() && time.Since(st.LastSelfUpdateCheck) < selfUpdateInterval {
		return selfUpdateSkipped
	}

	// 2. Save timestamp immediately to prevent retry storms.
	st.LastSelfUpdateCheck = time.Now()
	_ = state.Save(p.statePath, st)

	// 3. Fetch latest release version from GitHub API.
	latest, err := fetchLatestVersion(p.apiURL)
	if err != nil {
		output.Err("self-update failed (download error)")
		return selfUpdateFailed
	}

	// 4. Compare against current version -- if same or dev build, skip.
	currentVersion := strings.TrimPrefix(p.version, "v")
	if currentVersion == latest || currentVersion == "dev" {
		return selfUpdateSkipped
	}

	// 5. Download checksums and build archive URL.
	archiveName := fmt.Sprintf("tpack_%s_%s_%s.tar.gz", latest, runtime.GOOS, runtime.GOARCH)
	archiveURL := fmt.Sprintf("%s/v%s/%s", p.downloadURL, latest, archiveName)

	checksums, err := fetchChecksums(p.downloadURL, latest)
	if err != nil {
		output.Err("self-update failed (checksum fetch error)")
		return selfUpdateFailed
	}

	expectedHash, ok := checksums[archiveName]
	if !ok {
		output.Err("self-update failed (no checksum for archive)")
		return selfUpdateFailed
	}

	// 6. Download, verify integrity, and extract the new binary.
	newBinaryPath, cleanup, err := downloadVerifyExtract(archiveURL, expectedHash)
	if err != nil {
		output.Err("self-update failed (extract error)")
		return selfUpdateFailed
	}
	defer cleanup()

	// 7. Atomic replace: rename temp binary over current binary.
	if err := os.Rename(newBinaryPath, p.binaryPath); err != nil {
		output.Err(formatSelfUpdateReplaceError(err))
		return selfUpdateFailed
	}

	// 8. Git checkout matching tag to sync shell scripts.
	tag := "v" + latest
	if !p.skipGitSync {
		if err := syncGitRepo(p.repoDir, tag); err != nil {
			output.Warn("updated to " + tag + " (repo sync failed)")
			return selfUpdateSuccess
		}
	}

	// 9. Display success message.
	output.Ok("updated to " + tag)
	return selfUpdateSuccess
}

// Calls the GitHub API and returns the latest version
// without the "v" prefix.
func fetchLatestVersion(apiURL string) (string, error) {
	resp, cleanup, err := getResponse( //nolint:bodyclose // returned cleanup closes the response body
		apiURL,
		"application/vnd.github+json",
		"creating request",
		"fetching release",
		"unexpected status: ",
	)
	if err != nil {
		return "", err
	}
	defer cleanup()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// Downloads a tar.gz archive and extracts the Go binary
// to a temp directory. Returns the path to the extracted binary,
// a cleanup function, and any error.
func downloadAndExtract(url string) (string, func(), error) {
	resp, responseCleanup, err := getResponse( //nolint:bodyclose // returned cleanup closes the response body
		url,
		"",
		"creating request",
		"downloading archive",
		"download failed: status ",
	)
	if err != nil {
		return "", nil, err
	}
	defer responseCleanup()

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

// fetchChecksums downloads the checksums.txt file from a GitHub release and
// returns a map of filename to SHA-256 hex digest.
func fetchChecksums(baseURL, version string) (map[string]string, error) {
	url := fmt.Sprintf("%s/v%s/checksums.txt", baseURL, version)

	resp, cleanup, err := getResponse( //nolint:bodyclose // returned cleanup closes the response body
		url,
		"",
		"creating checksums request",
		"fetching checksums",
		"checksums download: status ",
	)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1 MiB limit
	if err != nil {
		return nil, fmt.Errorf("reading checksums: %w", err)
	}

	return parseChecksums(string(body)), nil
}

// parseChecksums parses a GoReleaser checksums.txt file.
// Each line has the format: "sha256hex  filename".
func parseChecksums(content string) map[string]string {
	sums := make(map[string]string)

	for _, line := range strings.Split(content, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			sums[parts[1]] = parts[0]
		}
	}

	return sums
}

// downloadVerifyExtract downloads a tar.gz archive, verifies its SHA-256
// checksum, and extracts the binary. Returns the path to the extracted binary,
// a cleanup function, and any error.
func downloadVerifyExtract(url, expectedHash string) (string, func(), error) {
	resp, responseCleanup, err := getResponse( //nolint:bodyclose // returned cleanup closes the response body
		url,
		"",
		"creating request",
		"downloading archive",
		"download failed: status ",
	)
	if err != nil {
		return "", nil, err
	}
	defer responseCleanup()

	tmpDir, err := os.MkdirTemp("", "tpack-update-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	// Save archive to temp file while computing checksum.
	archivePath := filepath.Join(tmpDir, "archive.tar.gz")

	archiveFile, err := os.Create(archivePath) //nolint:gosec // temp file in controlled directory
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("creating temp archive: %w", err)
	}

	hasher := sha256.New()

	if _, copyErr := io.Copy(archiveFile, io.TeeReader(resp.Body, hasher)); copyErr != nil {
		_ = archiveFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("saving archive: %w", copyErr)
	}

	_ = archiveFile.Close()

	// Verify checksum before extraction.
	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expectedHash {
		cleanup()
		return "", nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actual)
	}

	// Extract binary from verified archive.
	f, err := os.Open(archivePath)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	binaryPath, err := extractBinaryFromArchive(f, tmpDir)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}

func getResponse(url, accept, createError, fetchError, statusError string) (*http.Response, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("%s: %w", createError, err)
	}

	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // self-update URLs are supplied by trusted callers
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("%s: %w", fetchError, err)
	}

	cleanup := func() {
		_ = resp.Body.Close()
		cancel()
	}

	if resp.StatusCode != http.StatusOK {
		cleanup()
		return nil, nil, fmt.Errorf("%s%d", statusError, resp.StatusCode)
	}

	return resp, cleanup, nil
}

// Reads a gzip+tar stream and extracts the Go binary
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

// Runs `git checkout <tag>` in the given repository directory.
func syncGitRepo(repoDir, tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), selfUpdateTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", tag) //nolint:gosec // tag comes from GitHub API
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
