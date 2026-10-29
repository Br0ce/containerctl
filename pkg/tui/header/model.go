package header

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/tui/theme"
)

// Height is the number of lines the header occupies, including borders.
const Height = 4

// Model is the header with the Docker Engine API and Info boxes. It holds no
// client and performs no I/O.
type Model struct {
	host, version          string
	running, paused, total int
	width                  int
}

func New(host, version string) Model {
	return Model{host: host, version: version}
}

func (m Model) SetWidth(w int) Model {
	m.width = w
	return m
}

// SetShorts updates the container counts shown in the Info box.
func (m Model) SetShorts(shorts []container.Short) Model {
	m.running, m.paused = 0, 0
	for _, s := range shorts {
		switch s.State {
		case "running":
			m.running++
		case "paused":
			m.paused++
		}
	}
	m.total = len(shorts)
	return m
}

func (m Model) View() string {
	left := m.width / 2
	docker := box("Docker Engine API", left,
		field("Host:", 9, m.host),
		field("Version:", 9, m.version),
	)
	info := box("Info", m.width-left,
		field("Running", 8, fmt.Sprintf("%-6d", m.running))+"  "+field("Paused", 7, fmt.Sprint(m.paused)),
		field("Total", 8, fmt.Sprint(m.total)),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, docker, info)
}

func field(key string, keyWidth int, value string) string {
	return theme.Key.Width(keyWidth).Render(key) + theme.Value.Render(value)
}

// box puts lines in a theme box, padded by one column on each side. Lines that
// don't fit end in an ellipsis.
func box(title string, width int, lines ...string) string {
	inner := max(width-theme.BoxFrameWidth-2, 0)
	for i, l := range lines {
		lines[i] = " " + ansi.Truncate(l, inner, "…")
	}
	return theme.Box(title, width, strings.Join(lines, "\n"))
}
