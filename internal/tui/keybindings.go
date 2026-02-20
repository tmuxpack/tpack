package tui

import "github.com/charmbracelet/bubbles/key"

type sharedKeys struct {
	Quit      key.Binding
	ForceQuit key.Binding
	Back      key.Binding
}

type listKeys struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	Install   key.Binding
	Remove    key.Binding
	Update    key.Binding
	Clean     key.Binding
	Uninstall key.Binding
	Debug     key.Binding
	Browse    key.Binding
	Search    key.Binding
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
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
}

type browseKeys struct {
	Apply    key.Binding
	Cancel   key.Binding
	Filter   key.Binding
	Category key.Binding
	Open     key.Binding
}

type progressKeys struct {
	ViewCommits key.Binding
	BackToList  key.Binding
}

var BrowseKeys = browseKeys{
	Apply: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Category: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "category"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open"),
	),
}

var ProgressKeys = progressKeys{
	ViewCommits: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view commits"),
	),
	BackToList: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back to list"),
	),
}

var ListKeys = listKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab", " "),
		key.WithHelp("tab", "toggle"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove"),
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
	Browse: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "browse"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
}
