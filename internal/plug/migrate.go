package plug

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/tmuxpack/tpack/internal/git"
)

var errMigrationDestinationOccupied = errors.New("migration destination is occupied")

// MigrateLegacy moves matching legacy checkouts to their identity-based paths.
func MigrateLegacy(ctx context.Context, root Root, plugins []Plugin, origins git.OriginReader) error {
	rootPath, err := root.Path()
	if err != nil {
		return fmt.Errorf("migrate legacy plugins: %w", err)
	}
	rootInfo, err := os.Stat(rootPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect plugin root %s: %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("inspect plugin root %s: not a directory", rootPath)
	}

	lockPath := filepath.Join(rootPath, ".tpack-migrate.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open migration lock %s: %w", lockPath, err)
	}
	defer func() {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
	}()
	if err := lock.Chmod(0o600); err != nil {
		return fmt.Errorf("secure migration lock %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock plugin root %s: %w", rootPath, err)
	}

	for _, plugin := range plugins {
		if plugin.Alias != "" {
			continue
		}
		if err := migrateLegacyPlugin(ctx, root, plugin, origins); err != nil {
			return err
		}
	}
	return nil
}

func migrateLegacyPlugin(ctx context.Context, root Root, plugin Plugin, origins git.OriginReader) error {
	legacy, err := root.Child(LegacyPluginName(plugin.Spec))
	if err != nil {
		return migrationError(plugin, LegacyPluginName(plugin.Spec), plugin.DirName, "derive legacy path", err)
	}
	canonical, err := root.Child(plugin.DirName)
	if err != nil {
		return migrationError(plugin, legacy, plugin.DirName, "derive canonical path", err)
	}

	occupied, err := pathOccupied(canonical)
	if err != nil {
		return migrationError(plugin, legacy, canonical, "inspect destination", err)
	}
	if occupied {
		return nil
	}

	legacyInfo, err := os.Lstat(legacy)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return migrationError(plugin, legacy, canonical, "inspect source", err)
	}
	if legacyInfo.Mode()&os.ModeSymlink != 0 || !legacyInfo.IsDir() {
		return nil
	}

	origin, err := origins.Origin(ctx, legacy)
	if err != nil {
		return migrationError(plugin, legacy, canonical, "read Git origin", err)
	}
	originIdentity, err := NormalizeIdentity(origin)
	if err != nil {
		return migrationError(plugin, legacy, canonical, "normalize Git origin", err)
	}
	if originIdentity != plugin.Identity {
		return nil
	}

	occupied, err = pathOccupied(canonical)
	if err != nil {
		return migrationError(plugin, legacy, canonical, "recheck destination", err)
	}
	if occupied {
		return migrationError(plugin, legacy, canonical, "recheck destination", errMigrationDestinationOccupied)
	}
	if err := os.Rename(legacy, canonical); err != nil {
		return migrationError(plugin, legacy, canonical, "rename checkout", err)
	}
	return nil
}

func pathOccupied(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func migrationError(plugin Plugin, source, destination, operation string, err error) error {
	return fmt.Errorf("migrate plugin %q from %s to %s: %s: %w",
		plugin.Name, source, destination, operation, err)
}
