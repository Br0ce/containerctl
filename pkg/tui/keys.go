package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
)

var (
	quitKey = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	helpKey = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
)

// keyMap combines the global keys with those of the containers pane for the
// help footer.
type keyMap struct {
	table table.KeyMap
}

func (k keyMap) ShortHelp() []key.Binding {
	return append(k.table.ShortHelp(), quitKey, helpKey)
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{quitKey, helpKey},
		{k.table.LineUp, k.table.LineDown},
		{k.table.GotoTop, k.table.GotoBottom},
		{k.table.PageUp, k.table.PageDown},
		{k.table.HalfPageUp, k.table.HalfPageDown},
	}
}
