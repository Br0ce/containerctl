package files

import (
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/file"
	"github.com/Br0ce/containerctl/tui/theme"
)

// column is a table column. NAME takes the remaining space up to
// maxNameWidth; columns with a drop rank are hidden on narrow terminals,
// lowest rank first, so that NAME keeps at least minNameWidth.
type column struct {
	title string
	width int
	drop  int
}

const (
	minNameWidth = 24
	maxNameWidth = 40
)

var columns = []column{
	{title: "NAME"},
	{title: "PERMS", width: 10, drop: 2},    // "drwxr-xr-x"
	{title: "SIZE", width: 10},              // "1023.9 MB"
	{title: "MODIFIED", width: 12, drop: 1}, // "Oct 12 14:03"
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
		table.WithColumns(layout(0)),
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

	m.table.SetColumns(layout(innerW))
	m.table.SetWidth(innerW)
	m.width = w
	m.innerH = max(h-theme.BoxFrameHeight, 0)
	return m.fitTable()
}

// layout sizes the columns for the inner width w, giving hidden columns a
// width of 0, which the table skips.
func layout(w int) []table.Column {
	pad := theme.Cell.GetHorizontalPadding()
	cols := make([]table.Column, len(columns))
	for dropped := 0; ; dropped++ {
		rest := w
		for i, c := range columns {
			cols[i] = table.Column{Title: c.title}
			if c.drop > 0 && c.drop <= dropped {
				continue
			}
			cols[i].Width = c.width
			rest -= c.width + pad
		}
		cols[0].Width = min(max(rest, 0), maxNameWidth)
		if rest >= minNameWidth || dropped >= len(columns) {
			return cols
		}
	}
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
	now := time.Now()
	rows := make([]table.Row, len(files))
	for i, f := range files {
		rows[i] = row(f, now)
	}
	m.files = files
	m.path = path
	m.err = nil
	m.table.SetRows(rows)
	m.table.SetCursor(0)
	return m.fitTable()
}

// row renders f with one cell per column. The parent entry ".." shows only
// its name.
func row(f file.Info, now time.Time) table.Row {
	if f.Name == ".." {
		return table.Row{theme.DirCell(f.Name), "", "", ""}
	}
	name := theme.FileCell(f.Name)
	switch {
	case f.IsDir:
		name = theme.DirCell(f.Name)
	case f.IsSymlink():
		name = theme.FileCell(f.Name + " -> " + f.LinkTarget)
	}
	return table.Row{name, f.PermString(), f.SizeString(), f.ModTimeString(now)}
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
