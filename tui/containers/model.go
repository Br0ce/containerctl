package containers

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/tui/theme"
)

// Widths of the fixed columns. NAME and IMAGE share the remaining space on
// resize but never shrink below minFlexWidth.
const (
	idWidth      = 12
	statusWidth  = 20
	stateWidth   = 10
	minFlexWidth = 12
)

var columns = []table.Column{
	{Title: "ID", Width: idWidth},
	{Title: "NAME", Width: minFlexWidth},
	{Title: "IMAGE", Width: minFlexWidth},
	{Title: "STATUS", Width: statusWidth},
	{Title: "STATE", Width: stateWidth},
}

// Model is the containers pane. It holds no client and performs no I/O.
type Model struct {
	table  table.Model
	shorts []container.Short
	err    error
	width  int
	innerH int
}

func New() Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithKeyMap(keyMap()),
		table.WithStyles(table.Styles{
			Header:   theme.Header,
			Cell:     theme.Cell,
			Selected: theme.Selected,
		}),
	)
	return Model{table: t}
}

// keyMap limits paging to keys that don't collide with container actions
// (the default map uses f, u, d, b and space).
func keyMap() table.KeyMap {
	km := table.DefaultKeyMap()
	km.PageUp = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down"))
	km.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down"))
	return km
}

// SetSize sets the outer size of the pane, including its border.
func (m Model) SetSize(w, h int) Model {
	innerW := max(w-theme.BoxFrameWidth, 0)

	cols := make([]table.Column, len(columns))
	copy(cols, columns)
	flex := innerW - idWidth - statusWidth - stateWidth - len(cols)*theme.Cell.GetHorizontalPadding()
	cols[1].Width = max(flex/2, minFlexWidth)
	cols[2].Width = max(flex-flex/2, minFlexWidth)

	m.table.SetColumns(cols)
	m.table.SetWidth(innerW)
	m.width = w
	m.innerH = max(h-theme.BoxFrameHeight, 0)
	return m.fitTable()
}

// fitTable leaves room for the error line below the table.
func (m Model) fitTable() Model {
	h := m.innerH
	if m.err != nil {
		h = max(h-1, 0)
	}
	m.table.SetHeight(h)
	return m
}

// SetErr shows err below the table, or hides the error line if err is nil.
func (m Model) SetErr(err error) Model {
	m.err = err
	return m.fitTable()
}

// SetShorts replaces the rows and keeps the cursor on the same container
// across refreshes; the table itself only keeps the row index.
func (m Model) SetShorts(shorts []container.Short) Model {
	selID := ""
	if s, ok := m.Selected(); ok {
		selID = s.ID
	}

	rows := make([]table.Row, len(shorts))
	cursor := 0
	for i, s := range shorts {
		rows[i] = table.Row{shortID(s.ID), s.Name, s.Image, s.Status, theme.StateCell(s.State)}
		if s.ID == selID {
			cursor = i
		}
	}
	m.shorts = shorts
	m.table.SetRows(rows)
	m.table.SetCursor(cursor)
	return m
}

// Selected returns the container under the cursor.
func (m Model) Selected() (container.Short, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.shorts) {
		return container.Short{}, false
	}
	return m.shorts[i], true
}

// KeyMap returns the pane's key bindings for the help footer.
func (m Model) KeyMap() table.KeyMap {
	return m.table.KeyMap
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	content := m.table.View()
	if m.err != nil {
		content += "\n" + "Error: " + m.err.Error()
	}
	return theme.Box("Containers", m.width, content)
}

func shortID(id string) string {
	return id[:min(12, len(id))]
}
