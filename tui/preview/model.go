package preview

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/file"
	"github.com/Br0ce/containerctl/tui/theme"
)

const (
	// tabWidth is the distance between tab stops, as in most terminals.
	tabWidth = 8
	// hexBytes bounds the bytes of a binary file shown as a hex dump.
	hexBytes = 4096
	// control replaces control characters, which could otherwise move the
	// cursor or change the terminal's state.
	control = '·'
)

// KeyMap extends the viewport keys with jumps to the start and end.
type KeyMap struct {
	viewport.KeyMap
	Top    key.Binding
	Bottom key.Binding
}

// Model is the preview pane of the files page. It holds no client and
// performs no I/O; the content is set by the caller as it arrives.
type Model struct {
	viewport viewport.Model
	keys     KeyMap
	file     file.Info
	lines    []string
	loading  bool
	err      error
	focused  bool
	// spinner is the frame shown in front of the title while loading.
	spinner string
	width   int
}

func New() Model {
	keys := keyMap()
	vp := viewport.New()
	vp.KeyMap = keys.KeyMap
	return Model{viewport: vp, keys: keys}
}

// keyMap limits paging and horizontal scrolling to keys that don't collide
// with container actions, like the inspect pane does.
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
	m.viewport.SetWidth(max(w-theme.BoxFrameWidth, 0))
	m.viewport.SetHeight(max(h-theme.BoxFrameHeight, 0))
	m.width = w
	return m
}

// SetFocused shows the border in the accent color if the pane has the focus,
// and in gray if the files list next to it has.
func (m Model) SetFocused(focused bool) Model {
	m.focused = focused
	return m
}

// SetSpinner shows frame in front of the title, for a file that is loading;
// an empty frame hides it.
func (m Model) SetSpinner(frame string) Model {
	m.spinner = frame
	return m
}

// Loading reports whether the pane waits for the content of its file.
func (m Model) Loading() bool {
	return m.loading
}

// Start clears the pane for file f, which is loading.
func (m Model) Start(f file.Info) Model {
	m.file = f
	m.lines = nil
	m.loading = true
	m.err = nil
	m.viewport.SetXOffset(0)
	m.viewport.SetYOffset(0)
	return m.render()
}

// SetContent shows the first bytes of the file; truncated reports that the
// file is longer.
func (m Model) SetContent(data []byte, truncated bool) Model {
	m.lines = render(data, truncated)
	m.loading = false
	m.err = nil
	return m.render()
}

// SetErr shows err in place of the content.
func (m Model) SetErr(err error) Model {
	m.lines = nil
	m.loading = false
	m.err = err
	return m.render()
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
	title := "Preview"
	if m.file.Name != "" {
		title = m.file.Name
	}
	if m.spinner != "" {
		title = m.spinner + " " + title
	}
	if m.focused {
		return theme.Box(title, m.width, m.viewport.View())
	}
	return theme.DimBox(title, m.width, m.viewport.View())
}

func (m Model) render() Model {
	switch {
	case m.err != nil:
		m.viewport.SetContentLines([]string{" " + theme.Alert.Render("Error: "+m.err.Error())})
	case m.loading:
		m.viewport.SetContentLines([]string{" " + theme.Dim("Loading…")})
	default:
		m.viewport.SetContentLines(m.lines)
	}
	return m
}

// render lays out data as text, or, if it isn't text, as a note on its type
// followed by a hex dump. Images are only described for now.
func render(data []byte, truncated bool) []string {
	var lines []string
	text := data
	if truncated {
		text = trimPartialRune(text)
	}
	switch kind := http.DetectContentType(data); {
	case len(data) == 0:
		return []string{" " + theme.Dim("Empty file")}
	case isText(text):
		lines = indent(textLines(string(text)))
	case strings.HasPrefix(kind, "image/"):
		return []string{" " + theme.Dim("Image ("+kind+"), no preview yet")}
	default:
		lines = append([]string{" " + theme.Dim("Binary file ("+kind+")"), ""}, indent(hexLines(data))...)
	}
	if truncated {
		lines = append(lines, "", " "+theme.Dim("… truncated after "+theme.Bytes(uint64(len(data)))))
	}
	return lines
}

// isText reports whether b is UTF-8 without NUL bytes, which text files
// never contain.
func isText(b []byte) bool {
	return utf8.Valid(b) && !bytes.Contains(b, []byte{0})
}

// trimPartialRune drops an incomplete UTF-8 sequence at the end of b, which a
// truncated read may have cut in half.
func trimPartialRune(b []byte) []byte {
	for i := 1; i < utf8.UTFMax && i <= len(b); i++ {
		if utf8.RuneStart(b[len(b)-i]) {
			if !utf8.FullRune(b[len(b)-i:]) {
				return b[:len(b)-i]
			}
			break
		}
	}
	return b
}

// textLines splits s into lines that are safe to print: escape sequences are
// stripped, other control characters replaced and tabs expanded. A file in a
// container is untrusted, and raw escape sequences would reach the terminal.
func textLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = sanitize(ansi.Strip(l))
	}
	return lines
}

// sanitize replaces control characters in line and expands its tabs.
func sanitize(line string) string {
	var b strings.Builder
	// The column only matters for the next tab stop.
	tabs := strings.ContainsRune(line, '\t')
	col := 0
	for _, r := range line {
		switch {
		case r == '\t':
			n := tabWidth - col%tabWidth
			b.WriteString(strings.Repeat(" ", n))
			col += n
		case unicode.IsControl(r):
			b.WriteRune(control)
			col++
		default:
			b.WriteRune(r)
			if tabs {
				col += ansi.StringWidth(string(r))
			}
		}
	}
	return b.String()
}

// indent moves lines one column away from the border, like the notes.
func indent(lines []string) []string {
	for i, l := range lines {
		lines[i] = " " + l
	}
	return lines
}

// hexLines dumps the first hexBytes of data like hexdump -C.
func hexLines(data []byte) []string {
	return strings.Split(strings.TrimSuffix(hex.Dump(data[:min(len(data), hexBytes)]), "\n"), "\n")
}
