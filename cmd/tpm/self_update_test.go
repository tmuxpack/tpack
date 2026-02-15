package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmux-plugins/tpm/internal/state"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// createTestArchive creates a tar.gz archive containing a "tpm-go" file
// with the given content.
func createTestArchive(t *testing.T, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "tpm-go",
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
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Save a recent timestamp.
	st := state.State{LastSelfUpdateCheck: time.Now()}
	if err := state.Save(statePath, st); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      "http://unused",
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
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

func TestSelfUpdateSkipsWhenAlreadyLatest(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API returning current version.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v1.2.3"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.2.3",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped, got %d", result)
	}
}

func TestSelfUpdateSkipsDevVersion(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API returning a real version.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v2.0.0"})
	}))
	defer apiServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "dev",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped for dev version, got %d", result)
	}
}

func TestSelfUpdateDownloadsNewVersion(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Create the existing binary file.
	binaryPath := filepath.Join(dir, "tpm-go")
	if err := os.WriteFile(binaryPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	newContent := "new-binary-v2.0.0"
	archive := createTestArchive(t, newContent)

	// Mock GitHub API returning newer version.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v2.0.0"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	// Mock download server serving the archive.
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(archive)
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  binaryPath,
		apiURL:      apiServer.URL,
		downloadURL: downloadServer.URL,
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateSuccess {
		t.Errorf("expected selfUpdateSuccess, got %d", result)
	}

	// Verify the binary was replaced.
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("failed to read updated binary: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("binary content = %q, want %q", string(data), newContent)
	}

	// Verify success message was displayed.
	found := false
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == "TPM: updated to v2.0.0" {
			found = true
		}
	}
	if !found {
		t.Error("expected success DisplayMessage 'TPM: updated to v2.0.0'")
	}
}

func TestFetchLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		tagName    string
		wantVer    string
		statusCode int
		wantErr    bool
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
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					release := githubRelease{TagName: tt.tagName}
					json.NewEncoder(w).Encode(release)
				}
			}))
			defer server.Close()

			ver, err := fetchLatestVersion(server.URL)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
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
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API that fails.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	// Verify the download error message was displayed.
	found := false
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == "TPM: self-update failed (download error)" {
			found = true
		}
	}
	if !found {
		t.Error("expected DisplayMessage 'TPM: self-update failed (download error)'")
	}
}

func TestSelfUpdateDisplaysExtractError(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API returning newer version.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v2.0.0"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	// Mock download server returning invalid data.
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not a valid tar.gz"))
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: downloadServer.URL,
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	// Verify the extract error message was displayed.
	found := false
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == "TPM: self-update failed (extract error)" {
			found = true
		}
	}
	if !found {
		t.Error("expected DisplayMessage 'TPM: self-update failed (extract error)'")
	}
}

func TestSelfUpdateDisplaysPermissionError(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	newContent := "new-binary"
	archive := createTestArchive(t, newContent)

	// Mock GitHub API returning newer version.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v2.0.0"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	// Mock download server serving the archive.
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	// Use a binary path in a non-existent directory to trigger rename failure.
	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "nonexistent", "subdir", "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: downloadServer.URL,
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateFailed {
		t.Errorf("expected selfUpdateFailed, got %d", result)
	}

	// Verify the permission error message was displayed.
	found := false
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == "TPM: self-update failed (permission error)" {
			found = true
		}
	}
	if !found {
		t.Error("expected DisplayMessage 'TPM: self-update failed (permission error)'")
	}
}

func TestSelfUpdateTimestampSavedBeforeCheck(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API that takes a while (but we only care about state).
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	before := time.Now()
	selfUpdateCheck(p, runner)

	// Verify the timestamp was saved.
	st := state.Load(statePath)
	if st.LastSelfUpdateCheck.IsZero() {
		t.Error("expected LastSelfUpdateCheck to be set")
	}
	if st.LastSelfUpdateCheck.Before(before) {
		t.Error("expected LastSelfUpdateCheck to be after test start")
	}
}

func TestSelfUpdateVersionWithVPrefix(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Mock GitHub API returning same version as current (with v prefix).
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		release := githubRelease{TagName: "v1.2.3"}
		json.NewEncoder(w).Encode(release)
	}))
	defer apiServer.Close()

	runner := tmux.NewMockRunner()

	// Current version has v prefix -- should still match.
	p := selfUpdateParams{
		statePath:   statePath,
		version:     "v1.2.3",
		binaryPath:  filepath.Join(dir, "tpm-go"),
		apiURL:      apiServer.URL,
		downloadURL: "http://unused",
		skipGitSync: true,
	}

	result := selfUpdateCheck(p, runner)
	if result != selfUpdateSkipped {
		t.Errorf("expected selfUpdateSkipped, got %d", result)
	}
}

func TestSelfUpdateIntegration(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")

	// Create the existing binary file.
	binaryPath := filepath.Join(dir, "tpm-go")
	if err := os.WriteFile(binaryPath, []byte("old-binary-v1"), 0o755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

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

	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(archive)
	}))
	defer downloadServer.Close()

	runner := tmux.NewMockRunner()

	p := selfUpdateParams{
		statePath:   statePath,
		version:     "1.0.0",
		binaryPath:  binaryPath,
		apiURL:      apiServer.URL,
		downloadURL: downloadServer.URL,
		skipGitSync: true,
	}

	// Run the self-update.
	result := selfUpdateCheck(p, runner)
	if result != selfUpdateSuccess {
		t.Fatalf("expected selfUpdateSuccess, got %d", result)
	}

	// Verify binary was replaced with new content.
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("failed to read binary: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("binary content = %q, want %q", string(data), newContent)
	}

	// Verify binary is executable.
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("failed to stat binary: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Error("expected binary to be executable")
	}

	// Verify success message.
	found := false
	for _, call := range runner.Calls {
		if call.Method == "DisplayMessage" && len(call.Args) > 0 && call.Args[0] == "TPM: updated to v3.1.0" {
			found = true
		}
	}
	if !found {
		t.Error("expected DisplayMessage 'TPM: updated to v3.1.0'")
	}

	// Verify state was saved.
	st := state.Load(statePath)
	if st.LastSelfUpdateCheck.IsZero() {
		t.Error("expected LastSelfUpdateCheck to be set")
	}

	// Run again -- should skip because timestamp was saved recently.
	runner2 := tmux.NewMockRunner()
	result2 := selfUpdateCheck(p, runner2)
	if result2 != selfUpdateSkipped {
		t.Errorf("second run: expected selfUpdateSkipped, got %d", result2)
	}
}

func TestDownloadAndExtract(t *testing.T) {
	content := "test-binary-content"
	archive := createTestArchive(t, content)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	}))
	defer server.Close()

	binaryPath, cleanup, err := downloadAndExtract(server.URL)
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

	// Verify file name is tpm-go.
	if filepath.Base(binaryPath) != "tpm-go" {
		t.Errorf("base name = %q, want %q", filepath.Base(binaryPath), "tpm-go")
	}
}

func TestDownloadAndExtractServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, _, err := downloadAndExtract(server.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestDownloadAndExtractInvalidArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not a valid archive"))
	}))
	defer server.Close()

	_, _, err := downloadAndExtract(server.URL)
	if err == nil {
		t.Error("expected error for invalid archive")
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

	if hdr.Name != "tpm-go" {
		t.Errorf("header name = %q, want %q", hdr.Name, "tpm-go")
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
