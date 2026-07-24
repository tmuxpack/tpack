// Package plug provides the plugin model and parsing for tpack.
package plug

// Plugin represents a tmux plugin definition.
type Plugin struct {
	// Raw is the original plugin specification string (e.g. "user/repo#branch").
	Raw string
	// Name is the repository-qualified display and command name (e.g. "owner/repo").
	Name string
	// Identity is the normalized repository identity.
	Identity string
	// DirName is the validated directory name used to store the plugin.
	DirName string
	// Spec is the plugin specifier without branch (e.g. "user/repo" or full URL).
	// This may be a shorthand that requires NormalizeURL before cloning.
	Spec string
	// Branch is the optional branch to check out (empty string = default).
	Branch string
	// Alias is the optional directory override from "alias=X" in config.
	// When set, it overrides DirName without changing Name or Identity.
	Alias string
}

// LoadFailure records a plugin whose *.tmux failed to execute, with the
// captured error message.
type LoadFailure struct {
	Name    string `yaml:"name"`
	DirName string `yaml:"dir_name,omitempty"`
	Message string `yaml:"message"`
}
