package files

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/file"
	"github.com/Br0ce/containerctl/tui/theme"
)

// sizeWidth fits the longest size string, e.g. "1023.9 MB". NAME takes the
// remaining space.
const sizeWidth = 10

var columns = []table.Column{
	{Title: "NAME"},
	{Title: "SIZE", Width: sizeWidth},
}

// Model is the files pane. It holds no client and performs no I/O; listings
// are set by the caller as they arrive.
type Model struct {
	table  table.Model
	files  []file.Info
	path   string
	err    error
	width  int
	innerH int
}

func New() Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithKeyMap(keyMap()),
		table.WithStyles(styles(theme.Header, theme.Selected)),
	)
	return Model{table: t}
}

func styles(header, selected lipgloss.Style) table.Styles {
	return table.Styles{
		Header:   header,
		Cell:     theme.Cell,
		Selected: selected,
	}
}

// SetDark adapts the column headers and the selection to a dark or light
// terminal background.
func (m Model) SetDark(dark bool) Model {
	m.table.SetStyles(styles(theme.HeaderFor(dark), theme.SelectedFor(dark)))
	return m
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
	cols[0].Width = max(innerW-sizeWidth-len(cols)*theme.Cell.GetHorizontalPadding(), 0)

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

// Start clears the pane for a new container.
func (m Model) Start() Model {
	return m.SetFiles("", nil)
}

// SetFiles replaces the listing with the contents of directory path and
// clears any error.
func (m Model) SetFiles(path string, files []file.Info) Model {
	rows := make([]table.Row, len(files))
	for i, f := range files {
		name := theme.FileCell(f.Name)
		if f.IsDir {
			name = theme.DirCell(f.Name)
		}
		rows[i] = table.Row{name, f.SizeString()}
	}
	m.files = files
	m.path = path
	m.err = nil
	m.table.SetRows(rows)
	m.table.SetCursor(0)
	return m.fitTable()
}

// SetErr shows err below the listing, which is kept so the user can still
// navigate away from a directory that can't be read.
func (m Model) SetErr(err error) Model {
	m.err = err
	return m.fitTable()
}

// Selected returns the file under the cursor.
func (m Model) Selected() (file.Info, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.files) {
		return file.Info{}, false
	}
	return m.files[i], true
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
	title := "Files"
	if m.path != "" {
		title = m.path
	}
	content := m.table.View()
	if m.err != nil {
		content += "\n" + "Error: " + m.err.Error()
	}
	return theme.Box(title, m.width, content)
}
