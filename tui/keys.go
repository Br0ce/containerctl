package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"

	"github.com/Br0ce/containerctl/tui/inspect"
	"github.com/Br0ce/containerctl/tui/logs"
)

var (
	quitKey    = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	helpKey    = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	logsKey    = key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs"))
	filesKey   = key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "files"))
	inspectKey = key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect"))
	statsKey   = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "stats"))
	refreshKey = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh"))
	termKey    = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "terminal"))
	startKey   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start"))
	stopKey    = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop"))
	restartKey = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart"))
	removeKey  = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove"))
	confirmKey = key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm"))
	pauseKey   = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause"))
	unpauseKey = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unpause"))
	openKey    = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open dir"))
	backKey    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
)

// containersKeyMap combines the global keys with those of the containers pane
// for the help footer.
type containersKeyMap struct {
	table table.KeyMap
}

// ShortHelp leaves out the cursor keys, which are in the full help. On narrow
// terminals the footer cuts off the last keys with an ellipsis.
func (k containersKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{logsKey, inspectKey, statsKey, filesKey, termKey, startKey, stopKey, restartKey, pauseKey, unpauseKey, removeKey, quitKey, helpKey}
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k containersKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{logsKey, inspectKey},
		{statsKey, filesKey},
		{termKey},
		{startKey, stopKey},
		{pauseKey, unpauseKey},
		{restartKey, removeKey},
		{k.table.LineUp, k.table.LineDown},
		{k.table.GotoTop, k.table.GotoBottom},
		{k.table.PageUp, k.table.PageDown},
		{k.table.HalfPageUp, k.table.HalfPageDown},
		{quitKey, helpKey},
	}
}

// logsKeyMap combines the global keys with those of the logs pane for the help
// footer.
type logsKeyMap struct {
	logs logs.KeyMap
}

func (k logsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{backKey, k.logs.Up, k.logs.Down, quitKey, helpKey}
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k logsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{backKey},
		{k.logs.Up, k.logs.Down},
		{k.logs.Top, k.logs.Bottom},
		{k.logs.PageUp, k.logs.PageDown},
		{k.logs.HalfPageUp, k.logs.HalfPageDown},
		{k.logs.Left, k.logs.Right},
		{quitKey, helpKey},
	}
}

// filesKeyMap combines the global keys with those of the files pane for the
// help footer.
type filesKeyMap struct {
	table table.KeyMap
}

func (k filesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{openKey, backKey, k.table.LineUp, k.table.LineDown, quitKey, helpKey}
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k filesKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{openKey, backKey},
		{k.table.LineUp, k.table.LineDown},
		{k.table.GotoTop, k.table.GotoBottom},
		{k.table.PageUp, k.table.PageDown},
		{k.table.HalfPageUp, k.table.HalfPageDown},
		{quitKey, helpKey},
	}
}

// inspectKeyMap combines the global keys with those of the inspect pane for
// the help footer.
type inspectKeyMap struct {
	inspect inspect.KeyMap
}

func (k inspectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{backKey, refreshKey, k.inspect.Env, k.inspect.Up, k.inspect.Down, quitKey, helpKey}
}

// FullHelp pairs related bindings so the full help fits on two lines.
func (k inspectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{backKey, refreshKey},
		{k.inspect.Env},
		{k.inspect.Up, k.inspect.Down},
		{k.inspect.Top, k.inspect.Bottom},
		{k.inspect.PageUp, k.inspect.PageDown},
		{k.inspect.HalfPageUp, k.inspect.HalfPageDown},
		{k.inspect.Left, k.inspect.Right},
		{quitKey, helpKey},
	}
}

// statsKeyMap holds the global keys for the help footer of the stats pane,
// which has no keys of its own.
type statsKeyMap struct{}

func (k statsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{backKey, quitKey, helpKey}
}

func (k statsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{backKey},
		{quitKey, helpKey},
	}
}
