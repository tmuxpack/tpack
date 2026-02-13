package config

// ColorConfig holds optional color overrides from the config file.
// Empty strings mean "keep the existing value".
type ColorConfig struct {
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
	Accent    string `yaml:"accent"`
	Error     string `yaml:"error"`
	Muted     string `yaml:"muted"`
	Text      string `yaml:"text"`
}
