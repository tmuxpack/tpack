// Package config handles tpack configuration resolution.
package config

import "time"

const (
	// DefaultInstallKey is the default keybinding for plugin installation.
	DefaultInstallKey = "I"
	// DefaultUpdateKey is the default keybinding for plugin updates.
	DefaultUpdateKey = "U"
	// DefaultCleanKey is the default keybinding for plugin cleanup.
	DefaultCleanKey = "M-u"
	// DefaultTuiKey is the default keybinding for the TUI popup.
	DefaultTuiKey = "T"
	// DefaultPluginPath is the default plugin installation directory.
	DefaultPluginPath = ".tmux/plugins/"
	// PluginPathEnvVar is the current tmux environment variable for the plugin path.
	PluginPathEnvVar = "TPACK_PLUGIN_PATH"
	// LegacyPluginPathEnvVar is the legacy tmux environment variable for the plugin path.
	LegacyPluginPathEnvVar = "TMUX_PLUGIN_MANAGER_PATH"
	// SupportedTmuxVersion is the minimum tmux version encoded as major*100+minor.
	SupportedTmuxVersion = 109

	// Current tmux option names for keybinding customization.
	InstallKeyOption = "@tpack-install"
	UpdateKeyOption  = "@tpack-update"
	CleanKeyOption   = "@tpack-clean"
	TuiKeyOption     = "@tpack-tui"

	// Legacy tmux option names for keybinding customization (backwards compat).
	LegacyInstallKeyOption = "@tpm-install"
	LegacyUpdateKeyOption  = "@tpm-update"
	LegacyCleanKeyOption   = "@tpm-clean"

	// Current tmux option names for color overrides.
	ColorPrimaryOption   = "@tpack-color-primary"
	ColorSecondaryOption = "@tpack-color-secondary"
	ColorAccentOption    = "@tpack-color-accent"
	ColorErrorOption     = "@tpack-color-error"
	ColorMutedOption     = "@tpack-color-muted"
	ColorTextOption      = "@tpack-color-text"

	// Current tmux option names for update settings.
	UpdateIntervalOption = "@tpack-update-interval"
	UpdateModeOption     = "@tpack-update-mode"

	// VersionOption is the tmux option for pinning the tpack version.
	VersionOption = "@tpack-version"

	// AutoDownloadEnvVar is the current env var to opt out of auto-download.
	AutoDownloadEnvVar = "TPACK_AUTO_DOWNLOAD"
	// LegacyAutoDownloadEnvVar is the legacy env var to opt out of auto-download.
	LegacyAutoDownloadEnvVar = "TPM_AUTO_DOWNLOAD"
)

// Config holds resolved tpack configuration.
type Config struct {
	// PluginPath is the absolute path where plugins are installed.
	PluginPath string
	// TmuxConf is the path to the user's tmux.conf.
	TmuxConf string
	// InstallKey is the keybinding for plugin installation.
	InstallKey string
	// UpdateKey is the keybinding for plugin updates.
	UpdateKey string
	// CleanKey is the keybinding for plugin cleanup.
	CleanKey string
	// TuiKey is the keybinding for the TUI popup.
	TuiKey string
	// Colors holds optional color overrides from tmux options.
	Colors ColorConfig
	// UpdateCheckInterval is how often to check for plugin updates.
	UpdateCheckInterval time.Duration
	// UpdateMode controls update behavior ("auto", "prompt", or "off").
	UpdateMode string
	// PinnedVersion is the pinned tpack version from @tpack-version (empty = auto-update).
	PinnedVersion string
	// StatePath is the directory for persistent state (e.g. last update check).
	StatePath string
	// Home is the user's home directory, resolved during config resolution.
	Home string
}
