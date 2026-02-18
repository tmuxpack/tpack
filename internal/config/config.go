package config

import "time"

const (
	// Default keybinds
	DefaultInstallKey = "I"
	DefaultUpdateKey  = "U"
	DefaultCleanKey   = "M-u"
	DefaultTuiKey     = "T"

	DefaultPluginPath      = ".tmux/plugins/"
	PluginPathEnvVar       = "TPACK_PLUGIN_PATH"
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
	// Absolute path where plugins are installed.
	PluginPath string
	// User's tmux.conf.
	TmuxConf string

	// Keybinds
	InstallKey string
	UpdateKey  string
	CleanKey   string
	TuiKey     string

	// Color overrides from tmux options.
	Colors ColorConfig
	// How often to check for plugin updates.
	UpdateCheckInterval time.Duration
	// Controls update behavior ("auto", "prompt", or "off").
	UpdateMode string
	// PinnedVersion is the pinned tpack version from @tpack-version (empty = auto-update).
	PinnedVersion string
	// Directory for persistent state (e.g. last update check).
	StatePath string
	// User's home directory
	Home string
}
