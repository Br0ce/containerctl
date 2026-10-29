package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"

	"github.com/Br0ce/containerctl/pkg/tui/logs"
)

var (
	quitKey = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	helpKey = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	logsKey = key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs"))
	backKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
)

// containersKeyMap combines the global keys with those of the containers pane
// for the help footer.
type containersKeyMap struct {
	table table.KeyMap
}

func (k containersKeyMap) ShortHelp() []key.Binding {
	return append(k.table.ShortHelp(), logsKey, quitKey, helpKey)
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k containersKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{quitKey, helpKey},
		{logsKey},
		{k.table.LineUp, k.table.LineDown},
		{k.table.GotoTop, k.table.GotoBottom},
		{k.table.PageUp, k.table.PageDown},
		{k.table.HalfPageUp, k.table.HalfPageDown},
	}
}

// logsKeyMap combines the global keys with those of the logs pane for the help
// footer.
type logsKeyMap struct {
	logs logs.KeyMap
}

func (k logsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.logs.Up, k.logs.Down, backKey, quitKey, helpKey}
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k logsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{quitKey, helpKey},
		{backKey},
		{k.logs.Up, k.logs.Down},
		{k.logs.Top, k.logs.Bottom},
		{k.logs.PageUp, k.logs.PageDown},
		{k.logs.HalfPageUp, k.logs.HalfPageDown},
		{k.logs.Left, k.logs.Right},
	}
}
