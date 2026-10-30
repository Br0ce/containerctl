package containers

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/tui/theme"
)

// Widths of the fixed columns. COMPOSE, NAME, IMAGE and PORTS share the
// remaining space on resize but never shrink below minFlexWidth.
const (
	stateWidth   = 1
	idWidth      = 12
	healthWidth  = 6  // "HEALTH"
	statusWidth  = 16 // "Restarting (137)"
	ageWidth     = 4  // "11mo"
	minFlexWidth = 7
)

// The state is shown as a colored dot only, since STATUS already spells it
// out.
var columns = []table.Column{
	{Title: "", Width: stateWidth},
	{Title: "ID", Width: idWidth},
	{Title: "COMPOSE", Width: minFlexWidth},
	{Title: "NAME", Width: minFlexWidth},
	{Title: "IMAGE", Width: minFlexWidth},
	{Title: "HEALTH", Width: healthWidth},
	{Title: "STATUS", Width: statusWidth},
	{Title: "AGE", Width: ageWidth},
	{Title: "PORTS", Width: minFlexWidth},
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
		table.WithStyles(styles(theme.Header)),
	)
	return Model{table: t}
}

func styles(header lipgloss.Style) table.Styles {
	return table.Styles{
		Header:   header,
		Cell:     theme.Cell,
		Selected: theme.Selected,
	}
}

// SetDark adapts the column headers to a dark or light terminal background.
func (m Model) SetDark(dark bool) Model {
	m.table.SetStyles(styles(theme.HeaderFor(dark)))
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
	pad := theme.Cell.GetHorizontalPadding()
	flex := innerW - stateWidth - idWidth - healthWidth - statusWidth - ageWidth - len(cols)*pad
	// COMPOSE is hidden if it doesn't fit; the table skips columns of zero
	// width, padding included.
	flexCols := []int{2, 3, 4, 8}
	if flex < len(flexCols)*minFlexWidth {
		cols[2].Width = 0
		flexCols = flexCols[1:]
		flex += pad
	}
	n := len(flexCols)
	for _, c := range flexCols {
		cols[c].Width = max(flex/n, minFlexWidth)
	}
	cols[8].Width = max(flex-(n-1)*(flex/n), minFlexWidth)

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
		rows[i] = row(s)
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

// row renders a container. Containers that aren't running are dimmed so that
// the running ones stand out; the state dot keeps its color either way.
// Health is only shown for running containers.
func row(s container.Short) table.Row {
	cells := []string{shortID(s.ID), s.Project, s.Name, s.Image, s.Status, s.Age, formatPorts(s.Ports)}
	health := ""
	if s.State == "running" {
		health = theme.HealthMark(s.Health)
	} else {
		for i, c := range cells {
			cells[i] = theme.Dim(c)
		}
	}
	return append(table.Row{theme.StateCell(s.State)}, slices.Insert(cells, 4, health)...)
}

// formatPorts renders ports as host→container, e.g. "8888→8080, 53/udp". The
// container port is left out if it equals the host port, the protocol unless
// it is udp or sctp.
func formatPorts(ports []container.Port) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		part := strconv.Itoa(int(p.Public))
		if p.Private != p.Public {
			part += "→" + strconv.Itoa(int(p.Private))
		}
		if p.Proto != "tcp" {
			part += "/" + p.Proto
		}
		parts[i] = part
	}
	return strings.Join(parts, ", ")
}

func shortID(id string) string {
	return id[:min(12, len(id))]
}
