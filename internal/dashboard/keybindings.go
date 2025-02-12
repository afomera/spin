package dashboard

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all the keyboard shortcuts for the dashboard
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Tab         key.Binding
	Restart     key.Binding
	Stop        key.Binding
	Debug       key.Binding
	Logs        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Search      key.Binding
	Escape      key.Binding
	Quit        key.Binding
	ToggleInput key.Binding
	Enter       key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Logs, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab},
		{k.PageUp, k.PageDown},
		{k.Restart, k.Stop},
		{k.Debug, k.Logs},
		{k.Search},
		{k.Quit},
	}
}

// DefaultKeyMap returns a KeyMap with default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch panel"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		Debug: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "debug"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "toggle logs"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search logs"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit search/input"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		ToggleInput: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "toggle input"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute command"),
		),
	}
}
