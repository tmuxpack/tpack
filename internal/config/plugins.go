package config

import (
	"context"
	"fmt"

	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// LoadPlugins gathers and validates the complete plugin configuration before
// migrating legacy checkouts to their repository-identity paths.
func LoadPlugins(
	ctx context.Context,
	runner tmux.Runner,
	fs FS,
	paths Paths,
	origins git.OriginReader,
	warn func(string),
) ([]plug.Plugin, error) {
	plugins, err := GatherPlugins(runner, fs, paths, warn)
	if err != nil {
		return nil, err
	}
	_, err = plug.MigrateLegacy(ctx, paths.PluginPath, plugins, origins)
	if err != nil {
		return nil, fmt.Errorf("migrate legacy plugins: %w", err)
	}
	return plugins, nil
}
