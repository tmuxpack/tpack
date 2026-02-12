// Package config handles TPM configuration resolution.
package config

const (
	// DefaultInstallKey is the default keybinding for plugin installation.
	DefaultInstallKey = "I"
	// DefaultUpdateKey is the default keybinding for plugin updates.
	DefaultUpdateKey = "U"
	// DefaultCleanKey is the default keybinding for plugin cleanup.
	DefaultCleanKey = "M-u"
	// DefaultTPMPath is the default plugin installation directory.
	DefaultTPMPath = ".tmux/plugins/"
	// TPMEnvVar is the tmux environment variable for the plugin path.
	TPMEnvVar = "TMUX_PLUGIN_MANAGER_PATH"
	// SupportedTmuxVersion is the minimum tmux version (as int digits).
	SupportedTmuxVersion = 19

	// Tmux option names for keybinding customization.
	InstallKeyOption = "@tpm-install"
	UpdateKeyOption  = "@tpm-update"
	CleanKeyOption   = "@tpm-clean"
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
}
