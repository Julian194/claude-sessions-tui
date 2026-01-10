package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings
type KeyMap struct {
	// Navigation
	Up     key.Binding
	Down   key.Binding
	Tab    key.Binding
	Filter key.Binding

	// Actions
	Select  key.Binding
	Export  key.Binding
	CopyMD  key.Binding
	Branch  key.Binding
	Refresh key.Binding
	Pin     key.Binding

	// Toggle
	ToggleActivity key.Binding

	// Quit
	Quit key.Binding
}

// DefaultKeyMap returns the default key bindings
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
			key.WithHelp("tab", "switch pane"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "resume"),
		),
		Export: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "export"),
		),
		CopyMD: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "copy md"),
		),
		Branch: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "branch"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		Pin: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pin"),
		),
		ToggleActivity: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "activity"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns a short help string
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Export, k.CopyMD, k.Branch, k.Pin, k.Refresh, k.Quit}
}

// FullHelp returns the full help for all bindings
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.Filter},
		{k.Select, k.Export, k.CopyMD, k.Branch},
		{k.Pin, k.Refresh, k.ToggleActivity, k.Quit},
	}
}
