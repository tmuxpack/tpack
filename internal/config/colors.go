package config

// ColorConfig holds optional color overrides from tmux options.
// Empty strings mean "keep the existing value".
type ColorConfig struct {
	Primary   string
	Secondary string
	Accent    string
	Error     string
	Muted     string
	Text      string
}
