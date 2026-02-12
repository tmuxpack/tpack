// Package tmux provides an interface for interacting with tmux.
package tmux

// Runner abstracts tmux server interactions for testability.
type Runner interface {
	// ShowOption returns the value of a tmux global option.
	// Equivalent to: tmux show-option -gqv <option>
	ShowOption(option string) (string, error)

	// ShowEnvironment returns the value of a tmux global environment variable.
	// Equivalent to: tmux show-environment -g <name>
	ShowEnvironment(name string) (string, error)

	// SetEnvironment sets a tmux global environment variable.
	// Equivalent to: tmux set-environment -g <name> <value>
	SetEnvironment(name, value string) error

	// BindKey binds a tmux key to run a shell command.
	// Equivalent to: tmux bind-key <key> run-shell <cmd>
	BindKey(key, cmd string) error

	// SourceFile sources a tmux configuration file.
	// Equivalent to: tmux source-file <path>
	SourceFile(path string) error

	// DisplayMessage shows a message in the tmux status line.
	// Equivalent to: tmux display-message <msg>
	DisplayMessage(msg string) error

	// RunShell runs a shell command inside tmux.
	// Equivalent to: tmux run-shell <cmd>
	RunShell(cmd string) error

	// CommandPrompt opens a tmux command prompt.
	// Equivalent to: tmux command-prompt -p <prompt> <template>
	CommandPrompt(prompt, template string) error

	// Version returns the raw tmux version string (e.g. "tmux 3.4").
	Version() (string, error)

	// StartServer ensures a tmux server is running.
	// Equivalent to: tmux start-server
	StartServer() error

	// ShowWindowOption returns the value of a global window option.
	// Equivalent to: tmux show -gw <option>
	ShowWindowOption(option string) (string, error)

	// SetOption sets a tmux global option.
	// Equivalent to: tmux set-option -gq <option> <value>
	SetOption(option, value string) error
}
