// Package plugin provides the plugin model and parsing for TPM.
package plugin

// Plugin represents a tmux plugin definition.
type Plugin struct {
	// Raw is the original plugin specification string (e.g. "user/repo#branch").
	Raw string
	// Name is the derived plugin name (e.g. "repo").
	Name string
	// Spec is the plugin specifier without branch (e.g. "user/repo" or full URL).
	// This may be a shorthand that requires NormalizeURL before cloning.
	Spec string
	// Branch is the optional branch to check out (empty string = default).
	Branch string
}
