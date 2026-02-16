package tui

import "github.com/charmbracelet/bubbles/key"

type sharedKeys struct {
	Quit      key.Binding
	ForceQuit key.Binding
}

type listKeys struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	Install   key.Binding
	Update    key.Binding
	Clean     key.Binding
	Uninstall key.Binding
	Debug     key.Binding
}

var SharedKeys = sharedKeys{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	ForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "force quit"),
	),
}

var ListKeys = listKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab", " "),
		key.WithHelp("tab", "toggle"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update"),
	),
	Clean: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clean"),
	),
	Uninstall: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "uninstall"),
	),
	Debug: key.NewBinding(
		key.WithKeys("@"),
	),
}
