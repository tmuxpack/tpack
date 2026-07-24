package plug_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
)

func TestMigrateLegacyMovesMatchingCheckout(t *testing.T) {
	rootPath := t.TempDir()
	root := mustRoot(t, rootPath)
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	mustMkdir(t, legacy)
	mustWriteFile(t, filepath.Join(legacy, "marker"), "catppuccin")
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	if err := plug.MigrateLegacy(context.Background(), root, []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(rootPath, p.DirName, "marker"))
	if err != nil || string(data) != "catppuccin" {
		t.Fatalf("migrated marker = %q, %v", data, err)
	}
	if _, lstatErr := os.Lstat(legacy); !os.IsNotExist(lstatErr) {
		t.Fatalf("legacy path remains: %v", lstatErr)
	}
	lockInfo, err := os.Stat(filepath.Join(rootPath, ".tpack-migrate.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if got := lockInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("lock mode = %o, want 600", got)
	}
}

func TestMigrateLegacyAcceptsEquivalentOriginURLForm(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "https://github.com/catppuccin/tmux.git")
	legacy := filepath.Join(rootPath, "tmux")
	mustMkdir(t, legacy)

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p},
		&git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"})
	if err != nil {
		t.Fatal(err)
	}
	assertPathExists(t, filepath.Join(rootPath, p.DirName))
}

func TestMigrateLegacyOriginMismatchIsNoOp(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	mustMkdir(t, legacy)

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p},
		&git.MockOriginReader{URL: "git@github.com:someone-else/tmux.git"})
	if err != nil {
		t.Fatal(err)
	}
	assertPathExists(t, legacy)
	assertPathMissing(t, filepath.Join(rootPath, p.DirName))
}

func TestMigrateLegacySkipsAlias(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux alias=theme")
	legacy := filepath.Join(rootPath, "tmux")
	mustMkdir(t, legacy)
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	assertPathExists(t, legacy)
	if len(origins.Calls) != 0 {
		t.Fatalf("origin calls = %v, want none", origins.Calls)
	}
}

func TestMigrateLegacySkipsOccupiedCanonicalPath(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)
	mustMkdir(t, canonical)
	mustWriteFile(t, filepath.Join(canonical, "marker"), "canonical")
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(canonical, "marker"), "canonical")
	assertPathExists(t, legacy)
	if len(origins.Calls) != 0 {
		t.Fatalf("origin calls = %v, want none", origins.Calls)
	}
}

func TestMigrateLegacyTreatsDanglingCanonicalSymlinkAsOccupied(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)
	if err := os.Symlink(filepath.Join(rootPath, "missing-target"), canonical); err != nil {
		t.Fatal(err)
	}
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(canonical)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("canonical symlink changed: %v, %v", info, err)
	}
	assertPathExists(t, legacy)
}

func TestMigrateLegacyDoesNotCreateMissingRoot(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "missing")
	p := mustParsePlugin(t, "catppuccin/tmux")

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, &git.MockOriginReader{}); err != nil {
		t.Fatal(err)
	}
	assertPathMissing(t, rootPath)
}

func TestMigrateLegacyMissingLegacyPathIsNoOp(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	origins := &git.MockOriginReader{}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	if len(origins.Calls) != 0 {
		t.Fatalf("origin calls = %v, want none", origins.Calls)
	}
}

func TestMigrateLegacyRestrictsExistingLockFileMode(t *testing.T) {
	rootPath := t.TempDir()
	lockPath := filepath.Join(rootPath, ".tpack-migrate.lock")
	mustWriteFile(t, lockPath, "")
	if err := os.Chmod(lockPath, 0o666); err != nil {
		t.Fatal(err)
	}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), nil, &git.MockOriginReader{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("lock mode = %o, want 600", got)
	}
}

func TestMigrateLegacyRejectsSymlinkLockWithoutTouchingTarget(t *testing.T) {
	rootPath := t.TempDir()
	target := filepath.Join(rootPath, "unrelated")
	mustWriteFile(t, target, "unchanged")
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(rootPath, ".tpack-migrate.lock")
	if err := os.Symlink(target, lockPath); err != nil {
		t.Fatal(err)
	}

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), nil, &git.MockOriginReader{})
	if err == nil {
		t.Error("expected symlink lock error")
	} else if !strings.Contains(err.Error(), lockPath) {
		t.Errorf("error = %q, want lock path %q", err, lockPath)
	}
	assertFileContent(t, target, "unchanged")
	info, statErr := os.Stat(target)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("target mode = %o, want 644", got)
	}
}

func TestMigrateLegacyRejectsNonRegularLockWithoutTouchingIt(t *testing.T) {
	rootPath := t.TempDir()
	lockPath := filepath.Join(rootPath, ".tpack-migrate.lock")
	mustMkdir(t, lockPath)
	if err := os.Chmod(lockPath, 0o751); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(lockPath, "marker")
	mustWriteFile(t, marker, "unchanged")

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), nil, &git.MockOriginReader{})
	if err == nil {
		t.Error("expected non-regular lock error")
	} else if !strings.Contains(err.Error(), lockPath) {
		t.Errorf("error = %q, want lock path %q", err, lockPath)
	}
	assertFileContent(t, marker, "unchanged")
	info, statErr := os.Stat(lockPath)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o751 {
		t.Errorf("lock directory mode = %o, want 751", got)
	}
}

func TestMigrateLegacyLeavesLegacySymlinkUntouched(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	target := filepath.Join(rootPath, "target")
	mustMkdir(t, target)
	legacy := filepath.Join(rootPath, "tmux")
	if err := os.Symlink(target, legacy); err != nil {
		t.Fatal(err)
	}
	origins := &git.MockOriginReader{}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(legacy)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("legacy symlink changed: %v, %v", info, err)
	}
	if len(origins.Calls) != 0 {
		t.Fatalf("origin calls = %v, want none", origins.Calls)
	}
}

func TestMigrateLegacySkipsLegacyNonDirectory(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	mustWriteFile(t, legacy, "not a checkout")

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, &git.MockOriginReader{}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, legacy, "not a checkout")
}

func TestMigrateLegacyIsIdempotent(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	mustMkdir(t, filepath.Join(rootPath, "tmux"))
	origins := &git.MockOriginReader{URL: "git@github.com:catppuccin/tmux.git"}
	root := mustRoot(t, rootPath)

	if err := plug.MigrateLegacy(context.Background(), root, []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	if err := plug.MigrateLegacy(context.Background(), root, []plug.Plugin{p}, origins); err != nil {
		t.Fatal(err)
	}
	assertPathExists(t, filepath.Join(rootPath, p.DirName))
	if len(origins.Calls) != 1 {
		t.Fatalf("origin calls = %v, want one", origins.Calls)
	}
}

func TestMigrateLegacySelectsMatchingSameBasenameDeclaration(t *testing.T) {
	rootPath := t.TempDir()
	first := mustParsePlugin(t, "first/tmux")
	second := mustParsePlugin(t, "second/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	mustMkdir(t, legacy)
	origins := &git.MockOriginReader{URL: "git@github.com:second/tmux.git"}

	if err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{first, second}, origins); err != nil {
		t.Fatal(err)
	}
	assertPathMissing(t, filepath.Join(rootPath, first.DirName))
	assertPathExists(t, filepath.Join(rootPath, second.DirName))
	if len(origins.Calls) != 2 {
		t.Fatalf("origin calls = %v, want two inspections", origins.Calls)
	}
}

func TestMigrateLegacyWrapsOriginFailure(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)
	wantErr := errors.New("origin failed")

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p},
		&git.MockOriginReader{Err: wantErr})
	assertMigrationError(t, err, wantErr, p.Name, legacy, canonical)
	assertPathExists(t, legacy)
}

func TestMigrateLegacyWrapsOriginNormalizationFailure(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p},
		&git.MockOriginReader{URL: "https:///missing-host"})
	if err == nil {
		t.Fatal("expected normalization error")
	}
	assertErrorContext(t, err, p.Name, legacy, canonical)
	assertPathExists(t, legacy)
}

func TestMigrateLegacyRejectsDestinationCreatedDuringOriginInspection(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)
	mustWriteFile(t, filepath.Join(legacy, "marker"), "legacy")
	origins := originReaderFunc(func(_ context.Context, _ string) (string, error) {
		mustMkdir(t, canonical)
		mustWriteFile(t, filepath.Join(canonical, "marker"), "destination")
		return "git@github.com:catppuccin/tmux.git", nil
	})

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins)
	if err == nil || !strings.Contains(err.Error(), "occupied") {
		t.Fatalf("error = %v, want occupied destination error", err)
	}
	assertErrorContext(t, err, p.Name, legacy, canonical)
	assertFileContent(t, filepath.Join(legacy, "marker"), "legacy")
	assertFileContent(t, filepath.Join(canonical, "marker"), "destination")
}

func TestMigrateLegacyWrapsRenameFailure(t *testing.T) {
	rootPath := t.TempDir()
	p := mustParsePlugin(t, "catppuccin/tmux")
	legacy := filepath.Join(rootPath, "tmux")
	canonical := filepath.Join(rootPath, p.DirName)
	mustMkdir(t, legacy)
	origins := originReaderFunc(func(_ context.Context, _ string) (string, error) {
		if err := os.Remove(legacy); err != nil {
			t.Fatal(err)
		}
		return "git@github.com:catppuccin/tmux.git", nil
	})

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{p}, origins)
	assertMigrationError(t, err, fs.ErrNotExist, p.Name, legacy, canonical)
}

func TestMigrateLegacyKeepsEarlierMigrationWhenLaterOriginFails(t *testing.T) {
	rootPath := t.TempDir()
	first := mustParsePlugin(t, "owner/one")
	second := mustParsePlugin(t, "owner/two")
	mustMkdir(t, filepath.Join(rootPath, "one"))
	mustMkdir(t, filepath.Join(rootPath, "two"))
	wantErr := errors.New("second origin failed")
	origins := originReaderFunc(func(_ context.Context, dir string) (string, error) {
		if filepath.Base(dir) == "two" {
			return "", wantErr
		}
		return "git@github.com:owner/one.git", nil
	})

	err := plug.MigrateLegacy(context.Background(), mustRoot(t, rootPath), []plug.Plugin{first, second}, origins)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped origin failure", err)
	}
	assertPathExists(t, filepath.Join(rootPath, first.DirName))
	assertPathMissing(t, filepath.Join(rootPath, "one"))
	assertPathExists(t, filepath.Join(rootPath, "two"))
}

type originReaderFunc func(context.Context, string) (string, error)

func (f originReaderFunc) Origin(ctx context.Context, dir string) (string, error) {
	return f(ctx, dir)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("content of %s = %q, want %q", path, got, want)
	}
}

func assertMigrationError(t *testing.T, err, cause error, context ...string) {
	t.Helper()
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped %v", err, cause)
	}
	assertErrorContext(t, err, context...)
}

func assertErrorContext(t *testing.T, err error, context ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, value := range context {
		if !strings.Contains(err.Error(), value) {
			t.Fatalf("error = %q, want context %q", err, value)
		}
	}
}
