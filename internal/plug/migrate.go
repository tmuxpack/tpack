package plug

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

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
	lock, err := openMigrationLock(lockPath)
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
	if err := acquireMigrationLock(ctx, lock); err != nil {
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

const migrationLockRetryInterval = 10 * time.Millisecond

func acquireMigrationLock(ctx context.Context, lock *os.File) error {
	retry := time.NewTicker(migrationLockRetryInterval)
	defer retry.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-retry.C:
		}
	}
}

func openMigrationLock(path string) (*os.File, error) {
	for {
		inspected, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			lock, openErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
			if errors.Is(openErr, os.ErrExist) {
				continue
			}
			if openErr != nil {
				return nil, openErr
			}
			if validateErr := validateMigrationLock(lock, path, nil); validateErr != nil {
				return nil, closeRejectedLock(lock, validateErr)
			}
			return lock, nil
		}
		if err != nil {
			return nil, err
		}
		if !inspected.Mode().IsRegular() {
			return nil, fmt.Errorf("lock path is not a regular file (mode %s)", inspected.Mode())
		}

		lock, openErr := os.OpenFile(path, os.O_RDWR, 0)
		if errors.Is(openErr, os.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return nil, openErr
		}
		if validateErr := validateMigrationLock(lock, path, inspected); validateErr != nil {
			return nil, closeRejectedLock(lock, validateErr)
		}
		return lock, nil
	}
}

func validateMigrationLock(lock *os.File, path string, inspected os.FileInfo) error {
	opened, err := lock.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened lock: %w", err)
	}
	if !opened.Mode().IsRegular() {
		return fmt.Errorf("opened lock is not a regular file (mode %s)", opened.Mode())
	}
	current, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("reinspect lock path: %w", err)
	}
	if !current.Mode().IsRegular() {
		return fmt.Errorf("lock path is not a regular file (mode %s)", current.Mode())
	}
	if inspected != nil && !os.SameFile(inspected, opened) {
		return errors.New("lock path changed while opening")
	}
	if !os.SameFile(opened, current) {
		return errors.New("opened lock does not match lock path")
	}
	return nil
}

func closeRejectedLock(lock *os.File, validationErr error) error {
	if err := lock.Close(); err != nil {
		return errors.Join(validationErr, fmt.Errorf("close rejected lock: %w", err))
	}
	return validationErr
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
