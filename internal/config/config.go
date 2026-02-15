// Package config handles TPM configuration resolution.
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
	// DefaultTPMPath is the default plugin installation directory.
	DefaultTPMPath = ".tmux/plugins/"
	// TPMEnvVar is the tmux environment variable for the plugin path.
	TPMEnvVar = "TMUX_PLUGIN_MANAGER_PATH"
	// SupportedTmuxVersion is the minimum tmux version encoded as major*100+minor.
	SupportedTmuxVersion = 109

	// Tmux option names for keybinding customization.
	InstallKeyOption = "@tpm-install"
	UpdateKeyOption  = "@tpm-update"
	CleanKeyOption   = "@tpm-clean"
	TuiKeyOption     = "@tpm-tui"

	// Tmux option names for color overrides.
	ColorPrimaryOption   = "@tpm-color-primary"
	ColorSecondaryOption = "@tpm-color-secondary"
	ColorAccentOption    = "@tpm-color-accent"
	ColorErrorOption     = "@tpm-color-error"
	ColorMutedOption     = "@tpm-color-muted"
	ColorTextOption      = "@tpm-color-text"

	// Tmux option names for update settings.
	UpdateIntervalOption = "@tpm-update-interval"
	UpdateModeOption     = "@tpm-update-mode"

	// VersionOption is the tmux option for pinning the tpm version.
	VersionOption = "@tpm-version"
)

// Config holds resolved TPM configuration.
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
	// PinnedVersion is the pinned tpm version from @tpm-version (empty = auto-update).
	PinnedVersion string
	// StatePath is the directory for persistent state (e.g. last update check).
	StatePath string
	// Home is the user's home directory, resolved during config resolution.
	Home string
}
