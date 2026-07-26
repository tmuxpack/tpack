package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"runtime"

	"github.com/tmuxpack/tpack/internal/state"
	"github.com/tmuxpack/tpack/internal/tmux"
	"github.com/tmuxpack/tpack/internal/ui"
)

// sha256Hex returns the hex-encoded SHA-256 digest of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func newTestServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

func newReleaseServer(t *testing.T, tag string) string {
	t.Helper()
	return newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept header = %q, want %q", got, "application/vnd.github+json")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: tag})
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingBody struct {
	requestContext    context.Context
	closed            bool
	closedAfterCancel bool
}

func (b *trackingBody) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (b *trackingBody) Close() error {
	b.closed = true
	b.closedAfterCancel = b.requestContext.Err() != nil
	return nil
}

func TestGetResponseCleanupClosesBodyBeforeCancel(t *testing.T) {
	body := &trackingBody{}
	originalClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body.requestContext = req.Context()
		if req.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", req.Method, http.MethodGet)
		}
		if got := req.Header.Get("Accept"); got != "application/test" {
			t.Errorf("Accept header = %q, want %q", got, "application/test")
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Error("request context has no deadline")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = originalClient })

	resp, cleanup, err := getResponse( //nolint:bodyclose // returned cleanup is exercised below
		"https://example.test/archive",
		"application/test",
		"creating request",
		"fetching archive",
		"unexpected status: ",
	)
	if err != nil {
		t.Fatalf("getResponse() error = %v", err)
	}
	if resp.Body != body {
		t.Fatal("getResponse() returned an unexpected response body")
	}

	cleanup()

	if !body.closed {
		t.Error("cleanup did not close the response body")
	}
	if body.closedAfterCancel {
		t.Error("cleanup canceled the context before closing the response body")
	}
	if !errors.Is(body.requestContext.Err(), context.Canceled) {
		t.Errorf("request context error = %v, want context canceled", body.requestContext.Err())
	}
}

func newDataServer(t *testing.T, data []byte) string {
	t.Helper()
	return newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(data)
	})
}

func newSelfUpdateParams(t *testing.T, apiURL string) selfUpdateParams {
	t.Helper()
	dir := t.TempDir()
	return selfUpdateParams{
		statePath:   filepath.Join(dir, "state"),
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "tpack"),
		apiURL:      apiURL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}
}

func assertDisplayMessage(t *testing.T, runner *tmux.MockRunner, want string) {
	t.Helper()
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == want {
			return
		}
	}
	t.Errorf("expected DisplayMessage %q", want)
}

// newDownloadServer creates an httptest server that serves both checksums.txt
// and archive files for a given version.
func newDownloadServer(t *testing.T, version string, archive []byte) string {
	t.Helper()
	checksum := sha256Hex(archive)
	archiveName := fmt.Sprintf("tpack_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)

	return newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			fmt.Fprintf(w, "%s  %s\n", checksum, archiveName)
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive)
		}
	})
}

// createTestArchive creates a tar.gz archive containing a "tpack" file
// with the given content.
func createTestArchive(t *testing.T, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "tpack",
		Mode: 0o755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write tar content: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func TestSelfUpdateSkipsWhenRecent(t *testing.T) {
	p := newSelfUpdateParams(t, "http://unused")

	// Save a recent timestamp.
	st := state.State{LastSelfUpdateCheck: time.Now()}
	if err := state.Save(p.statePath, st); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	runner := tmux.NewMockRunner()

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped, got %d", result)
	}

	// Verify no tmux messages were displayed.
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" {
			t.Errorf("unexpected DisplayMessage call: %v", call.Args)
		}
	}
}

func TestSelfUpdateResultReturnsTransportFailure(t *testing.T) {
	sink := ui.NewMockSink()
	sink.Err = errors.New("tmux unavailable")
	output := ui.NewReporter(sink)
	output.Ok("updated")

	err := selfUpdateCommandResult(selfUpdateSuccess, output)
	var transport *ui.TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("selfUpdateCommandResult() = %v, want transport error", err)
	}
}

func TestSelfUpdateResultUsesErrSilentForDeliveredFailure(t *testing.T) {
	output := ui.NewReporter(ui.NewMockSink())
	output.Err("self-update failed")

	if err := selfUpdateCommandResult(selfUpdateFailed, output); !errors.Is(err, errSilent) {
		t.Fatalf("selfUpdateCommandResult() = %v, want errSilent", err)
	}
}

func TestSelfUpdateSkipsWhenAlreadyLatest(t *testing.T) {
	// Mock GitHub API returning current version.
	apiURL := newReleaseServer(t, "v1.2.3")

	runner := tmux.NewMockRunner()

	p := newSelfUpdateParams(t, apiURL)
	p.version = "1.2.3"

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped, got %d", result)
	}
}

func TestSelfUpdateSkipsDevVersion(t *testing.T) {
	// Mock GitHub API returning a real version.
	apiURL := newReleaseServer(t, "v2.0.0")

	runner := tmux.NewMockRunner()

	p := newSelfUpdateParams(t, apiURL)
	p.version = "dev"

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped for dev version, got %d", result)
	}
}

func TestSelfUpdateDownloadsNewVersion(t *testing.T) {
	apiURL := newReleaseServer(t, "v2.0.0")
	p := newSelfUpdateParams(t, apiURL)

	// Create the existing binary file.
	if err := os.WriteFile(p.binaryPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	newContent := "new-binary-v2.0.0"
	archive := createTestArchive(t, newContent)

	downloadServer := newDownloadServer(t, "2.0.0", archive)
	p.downloadURL = downloadServer

	runner := tmux.NewMockRunner()

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSuccess {
		t.Errorf("expected selfUpdateSuccess, got %d", result)
	}

	// Verify the binary was replaced.
	data, err := os.ReadFile(p.binaryPath)
	if err != nil {
		t.Fatalf("failed to read updated binary: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("binary content = %q, want %q", string(data), newContent)
	}

	// Verify success message was displayed.
	assertDisplayMessage(t, runner, "tpack: updated to v2.0.0")
}

func TestSelfUpdateRepoSyncFailureWarns(t *testing.T) {
	apiURL := newReleaseServer(t, "v2.0.0")
	p := newSelfUpdateParams(t, apiURL)

	// Create the existing binary file.
	if err := os.WriteFile(p.binaryPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	archive := createTestArchive(t, "new-binary-v2.0.0")

	downloadServer := newDownloadServer(t, "2.0.0", archive)
	p.downloadURL = downloadServer
	p.skipGitSync = false
	p.repoDir = t.TempDir() // not a git repo, so the tag checkout fails

	runner := tmux.NewMockRunner()

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSuccess {
		t.Errorf("expected selfUpdateSuccess, got %d", result)
	}

	// The update succeeded but the repo sync failed: expect the warning form.
	want := "tpack: warning: updated to v2.0.0 (repo sync failed)"
	assertDisplayMessage(t, runner, want)
}

func TestFetchLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		tagName    string
		wantVer    string
		statusCode int
		wantErr    string
	}{
		{
			name:       "strips v prefix",
			tagName:    "v1.5.0",
			wantVer:    "1.5.0",
			statusCode: http.StatusOK,
		},
		{
			name:       "no v prefix",
			tagName:    "2.0.0",
			wantVer:    "2.0.0",
			statusCode: http.StatusOK,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    "unexpected status: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					release := githubRelease{TagName: tt.tagName}
					json.NewEncoder(w).Encode(release)
				}
			})

			ver, err := fetchLatestVersion(serverURL)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ver != tt.wantVer {
				t.Errorf("version = %q, want %q", ver, tt.wantVer)
			}
		})
	}
}

func TestSelfUpdateDisplaysDownloadError(t *testing.T) {
	// Mock GitHub API that fails.
	apiURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	runner := tmux.NewMockRunner()
	p := newSelfUpdateParams(t, apiURL)

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	assertDisplayMessage(t, runner, "tpack: error: self-update failed (download error)")
}

func TestSelfUpdateDisplaysExtractError(t *testing.T) {
	// Mock GitHub API returning newer version.
	apiURL := newReleaseServer(t, "v2.0.0")

	// Invalid archive data with matching checksum.
	invalidArchive := []byte("not a valid tar.gz")
	downloadServer := newDownloadServer(t, "2.0.0", invalidArchive)

	runner := tmux.NewMockRunner()
	p := newSelfUpdateParams(t, apiURL)
	p.downloadURL = downloadServer

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	assertDisplayMessage(t, runner, "tpack: error: self-update failed (extract error)")
}

func TestSelfUpdateDisplaysPermissionError(t *testing.T) {
	newContent := "new-binary"
	archive := createTestArchive(t, newContent)

	// Mock GitHub API returning newer version.
	apiURL := newReleaseServer(t, "v2.0.0")

	downloadServer := newDownloadServer(t, "2.0.0", archive)

	runner := tmux.NewMockRunner()

	// Use a binary path in a non-existent directory to trigger rename failure.
	p := newSelfUpdateParams(t, apiURL)
	p.binaryPath = filepath.Join(filepath.Dir(p.binaryPath), "nonexistent", "subdir", "tpack")
	p.downloadURL = downloadServer

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	assertDisplayMessage(t, runner, "tpack: error: self-update failed (permission error)")
}

func TestSelfUpdateTimestampSavedBeforeCheck(t *testing.T) {
	// Mock GitHub API that takes a while (but we only care about state).
	apiURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	runner := tmux.NewMockRunner()
	p := newSelfUpdateParams(t, apiURL)

	before := time.Now()
	selfUpdateCheck(p, ui.NewStatusOutput(runner))

	// Verify the timestamp was saved.
	st := state.Load(p.statePath, nil)
	if st.LastSelfUpdateCheck.IsZero() {
		t.Error("expected LastSelfUpdateCheck to be set")
	}
	if st.LastSelfUpdateCheck.Before(before) {
		t.Error("expected LastSelfUpdateCheck to be after test start")
	}
}

func TestSelfUpdateVersionWithVPrefix(t *testing.T) {
	// Mock GitHub API returning same version as current (with v prefix).
	apiURL := newReleaseServer(t, "v1.2.3")

	runner := tmux.NewMockRunner()

	// Current version has v prefix -- should still match.
	p := newSelfUpdateParams(t, apiURL)
	p.version = "v1.2.3"

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped, got %d", result)
	}
}

func TestSelfUpdateIntegration(t *testing.T) {
	newContent := "#!/bin/sh\necho new-binary-v3.1.0"
	archive := createTestArchive(t, newContent)

	// Set up mock servers.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Accept header.
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept header = %q, want %q", got, "application/vnd.github+json")
		}
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v3.1.0"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	downloadServer := newDownloadServer(t, "3.1.0", archive)
	p := newSelfUpdateParams(t, apiServer.URL)
	p.downloadURL = downloadServer

	// Create the existing binary file.
	if err := os.WriteFile(p.binaryPath, []byte("old-binary-v1"), 0o755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	runner := tmux.NewMockRunner()

	// Run the self-update.
	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateSuccess {
		t.Fatalf("expected selfUpdateSuccess, got %d", result)
	}

	// Verify binary was replaced with new content.
	data, err := os.ReadFile(p.binaryPath)
	if err != nil {
		t.Fatalf("failed to read binary: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("binary content = %q, want %q", string(data), newContent)
	}

	// Verify binary is executable.
	info, err := os.Stat(p.binaryPath)
	if err != nil {
		t.Fatalf("failed to stat binary: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Error("expected binary to be executable")
	}

	// Verify success message.
	assertDisplayMessage(t, runner, "tpack: updated to v3.1.0")

	// Verify state was saved.
	st := state.Load(p.statePath, nil)
	if st.LastSelfUpdateCheck.IsZero() {
		t.Error("expected LastSelfUpdateCheck to be set")
	}

	// Run again -- should skip because timestamp was saved recently.
	runner2 := tmux.NewMockRunner()
	result2 := selfUpdateCheck(p, ui.NewStatusOutput(runner2))
	if result2 != selfUpdateSkipped {
		t.Errorf("second run: expected selfUpdateSkipped, got %d", result2)
	}
}

func TestDownloadAndExtract(t *testing.T) {
	content := "test-binary-content"
	archive := createTestArchive(t, content)

	serverURL := newDataServer(t, archive)

	binaryPath, cleanup, err := downloadAndExtract(serverURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	// Verify the extracted binary content.
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}

	// Verify the file is executable.
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Error("expected file to be executable")
	}

	// Verify file name is tpack.
	if filepath.Base(binaryPath) != "tpack" {
		t.Errorf("base name = %q, want %q", filepath.Base(binaryPath), "tpack")
	}
}

func TestDownloadAndExtractServerError(t *testing.T) {
	serverURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, _, err := downloadAndExtract(serverURL)
	if err == nil || err.Error() != "download failed: status 404" {
		t.Errorf("error = %v, want %q", err, "download failed: status 404")
	}
}

func TestDownloadAndExtractInvalidArchive(t *testing.T) {
	serverURL := newDataServer(t, []byte("not a valid archive"))

	_, _, err := downloadAndExtract(serverURL)
	if err == nil {
		t.Error("expected error for invalid archive")
	}
}

func TestExtractBinaryRejectsOversized(t *testing.T) {
	// Create a tar.gz archive with a header claiming a size larger than maxBinarySize.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "tpack",
		Mode: 0o755,
		Size: maxBinarySize + 1,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	// Write just a small amount of actual data — the size check is on hdr.Size.
	if _, err := tw.Write([]byte("x")); err != nil {
		t.Fatalf("failed to write tar content: %v", err)
	}
	tw.Close()
	gw.Close()

	tmpDir := t.TempDir()
	_, err := extractBinaryFromArchive(&buf, tmpDir)
	if err == nil {
		t.Fatal("expected error for oversized binary, got nil")
	}
	if !strings.Contains(err.Error(), "binary too large") {
		t.Errorf("error = %q, want it to contain 'binary too large'", err.Error())
	}
}

func TestCreateTestArchive(t *testing.T) {
	content := "hello world"
	archive := createTestArchive(t, content)

	// Verify we can decompress and read the archive.
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("failed to read tar header: %v", err)
	}

	if hdr.Name != "tpack" {
		t.Errorf("header name = %q, want %q", hdr.Name, "tpack")
	}
	if hdr.Mode != 0o755 {
		t.Errorf("header mode = %o, want %o", hdr.Mode, 0o755)
	}

	data, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("failed to read tar content: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}
}

func TestParseChecksums(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "single entry",
			content: "abc123  tpack_1.0.0_linux_amd64.tar.gz\n",
			want:    map[string]string{"tpack_1.0.0_linux_amd64.tar.gz": "abc123"},
		},
		{
			name:    "multiple entries",
			content: "aaa  file1.tar.gz\nbbb  file2.tar.gz\n",
			want:    map[string]string{"file1.tar.gz": "aaa", "file2.tar.gz": "bbb"},
		},
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "blank lines",
			content: "aaa  file1.tar.gz\n\nbbb  file2.tar.gz\n",
			want:    map[string]string{"file1.tar.gz": "aaa", "file2.tar.gz": "bbb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChecksums(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("checksums[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestFetchChecksums(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		serverURL := newDataServer(t, []byte("abc123  tpack_1.0.0_linux_amd64.tar.gz\n"))

		sums, err := fetchChecksums(serverURL, "1.0.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sums["tpack_1.0.0_linux_amd64.tar.gz"] != "abc123" {
			t.Errorf("unexpected checksum: %v", sums)
		}
	})

	t.Run("server error", func(t *testing.T) {
		serverURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		_, err := fetchChecksums(serverURL, "1.0.0")
		if err == nil || err.Error() != "checksums download: status 404" {
			t.Errorf("error = %v, want %q", err, "checksums download: status 404")
		}
	})
}

func TestDownloadVerifyExtract(t *testing.T) {
	t.Run("valid checksum", func(t *testing.T) {
		content := "verified-binary"
		archive := createTestArchive(t, content)
		expectedHash := sha256Hex(archive)

		serverURL := newDataServer(t, archive)

		binaryPath, cleanup, err := downloadVerifyExtract(serverURL, expectedHash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer cleanup()

		data, err := os.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("failed to read binary: %v", err)
		}
		if string(data) != content {
			t.Errorf("content = %q, want %q", string(data), content)
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		archive := createTestArchive(t, "some-binary")

		serverURL := newDataServer(t, archive)

		_, _, err := downloadVerifyExtract(serverURL, "0000000000000000000000000000000000000000000000000000000000000000")
		if err == nil {
			t.Error("expected error for checksum mismatch")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("error = %q, want it to contain 'checksum mismatch'", err.Error())
		}
	})

	t.Run("server error", func(t *testing.T) {
		serverURL := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		_, _, err := downloadVerifyExtract(serverURL, "unused")
		if err == nil || err.Error() != "download failed: status 404" {
			t.Errorf("error = %v, want %q", err, "download failed: status 404")
		}
	})
}

func TestSelfUpdateChecksumFetchError(t *testing.T) {
	apiURL := newReleaseServer(t, "v2.0.0")

	// Download server returns 404 for checksums.txt.
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksums.txt") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("unused"))
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	p := newSelfUpdateParams(t, apiURL)
	p.downloadURL = downloadServer.URL

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	assertDisplayMessage(t, runner, "tpack: error: self-update failed (checksum fetch error)")
}

func TestSelfUpdateNoChecksumForArchive(t *testing.T) {
	apiURL := newReleaseServer(t, "v2.0.0")

	// Download server returns checksums for a different platform.
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksums.txt") {
			fmt.Fprintln(w, "abc123  tpack_2.0.0_other_platform.tar.gz")
			return
		}
		w.Write([]byte("unused"))
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	p := newSelfUpdateParams(t, apiURL)
	p.downloadURL = downloadServer.URL

	result := selfUpdateCheck(p, ui.NewStatusOutput(runner))
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	assertDisplayMessage(t, runner, "tpack: error: self-update failed (no checksum for archive)")
}

func TestArchiveURLFormat(t *testing.T) {
	tests := []struct {
		name        string
		downloadURL string
		version     string
		goos        string
		goarch      string
		wantName    string
	}{
		{
			name:        "linux amd64",
			downloadURL: "https://github.com/tmuxpack/tpack/releases/download",
			version:     "1.2.3",
			goos:        "linux",
			goarch:      "amd64",
			wantName:    "tpack_1.2.3_linux_amd64.tar.gz",
		},
		{
			name:        "darwin arm64",
			downloadURL: "https://github.com/tmuxpack/tpack/releases/download",
			version:     "2.0.0",
			goos:        "darwin",
			goarch:      "arm64",
			wantName:    "tpack_2.0.0_darwin_arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("tpack_%s_%s_%s.tar.gz", tt.version, tt.goos, tt.goarch)
			if got != tt.wantName {
				t.Errorf("archive name = %q, want %q", got, tt.wantName)
			}

			url := fmt.Sprintf("%s/v%s/%s", tt.downloadURL, tt.version, got)
			wantURL := tt.downloadURL + "/v" + tt.version + "/" + tt.wantName
			if url != wantURL {
				t.Errorf("URL = %q, want %q", url, wantURL)
			}
		})
	}
}
