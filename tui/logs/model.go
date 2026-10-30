package logs

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/tui/theme"
)

// maxLines bounds the lines kept per session; older lines are dropped first.
const maxLines = 5000

// KeyMap extends the viewport keys with jumps to the start and end.
type KeyMap struct {
	viewport.KeyMap
	Top    key.Binding
	Bottom key.Binding
}

// Model is the logs pane. It holds no client and performs no I/O; lines are
// appended by the caller as they arrive.
type Model struct {
	viewport viewport.Model
	keys     KeyMap
	lines    []string
	width    int
}

func New() Model {
	keys := keyMap()
	vp := viewport.New()
	vp.KeyMap = keys.KeyMap
	return Model{viewport: vp, keys: keys}
}

// keyMap limits paging and horizontal scrolling to keys that don't collide
// with container actions (the default map uses f, b, u, d, h, l and space).
func keyMap() KeyMap {
	km := viewport.DefaultKeyMap()
	km.PageUp = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down"))
	km.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down"))
	km.Left = key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "scroll left"))
	km.Right = key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "scroll right"))
	return KeyMap{
		KeyMap: km,
		Top:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/home", "go to start")),
		Bottom: key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G/end", "go to end")),
	}
}

// SetSize sets the outer size of the pane, including its border.
func (m Model) SetSize(w, h int) Model {
	follow := m.viewport.AtBottom()
	m.viewport.SetWidth(max(w-theme.BoxFrameWidth, 0))
	m.viewport.SetHeight(max(h-theme.BoxFrameHeight, 0))
	if follow {
		m.viewport.GotoBottom()
	}
	m.width = w
	return m
}

// Start clears the pane for a new log session.
func (m Model) Start() Model {
	m.lines = nil
	m.viewport.SetContentLines(nil)
	m.viewport.SetXOffset(0)
	m.viewport.SetYOffset(0)
	return m
}

// Append adds lines to the end of the log. The view follows new lines while
// it is scrolled to the bottom and otherwise stays where it is.
func (m Model) Append(lines ...string) Model {
	follow := m.viewport.AtBottom()
	for _, l := range lines {
		// A carriage return would move the cursor back over the box border.
		m.lines = append(m.lines, strings.ReplaceAll(l, "\r", ""))
	}
	dropped := max(len(m.lines)-maxLines, 0)
	m.lines = m.lines[dropped:]

	// The viewport normalizes the lines it is given in place, so it gets its
	// own copy.
	m.viewport.SetContentLines(slices.Clone(m.lines))
	if follow {
		m.viewport.GotoBottom()
	} else {
		m.viewport.SetYOffset(m.viewport.YOffset() - dropped)
	}
	return m
}

// KeyMap returns the pane's key bindings for the help footer.
func (m Model) KeyMap() KeyMap {
	return m.keys
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Top):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.viewport.GotoBottom()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return theme.Box("Logs", m.width, m.viewport.View())
}
