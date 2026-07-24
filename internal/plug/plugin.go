// Package plug provides the plugin model and parsing for tpack.
package plug

// Plugin represents a tmux plugin definition.
type Plugin struct {
	// Raw is the original plugin specification string (e.g. "user/repo#branch").
	Raw string
	// Name is the derived plugin name (e.g. "repo").
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
	// Alias is the optional alias from "alias=X" in config.
	// When set, Name is derived from Alias instead of the spec.
	Alias string
}

// LoadFailure records a plugin whose *.tmux failed to execute, with the
// captured error message.
type LoadFailure struct {
	Name    string `yaml:"name"`
	Message string `yaml:"message"`
}
