package header

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/engine"
	"github.com/Br0ce/containerctl/tui/theme"
)

// Height is the number of lines the header occupies, including borders.
const Height = 4

// Model is the header with the Docker Engine API and Info boxes. It holds no
// client and performs no I/O.
type Model struct {
	host, version                  string
	running, paused, exited, total int
	unhealthy, projects            int
	width                          int
	// container, if set, is shown in the Info box instead of the counts.
	container *container.Short
	// engine, once fetched, replaces the client API version in the Docker
	// Engine API box.
	engine *engine.Info
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
	m.running, m.paused, m.exited, m.unhealthy = 0, 0, 0, 0
	projects := map[string]bool{}
	for _, s := range shorts {
		switch s.State {
		case "running":
			m.running++
			if s.Health == "unhealthy" {
				m.unhealthy++
			}
		case "paused":
			m.paused++
		case "exited":
			m.exited++
		}
		if s.Project != "" {
			projects[s.Project] = true
		}
	}
	m.total = len(shorts)
	m.projects = len(projects)
	return m
}

// SetEngine shows the host's name, versions and system in the Docker Engine
// API box.
func (m Model) SetEngine(e engine.Info) Model {
	m.engine = &e
	return m
}

// SetContainer shows the name and image of s in the Info box instead of the
// container counts.
func (m Model) SetContainer(s container.Short) Model {
	m.container = &s
	return m
}

// ClearContainer shows the container counts in the Info box again.
func (m Model) ClearContainer() Model {
	m.container = nil
	return m
}

func (m Model) View() string {
	left := m.width / 2
	return lipgloss.JoinHorizontal(lipgloss.Top, m.docker(left), m.info(m.width-left))
}

func (m Model) docker(width int) string {
	if m.engine == nil {
		return box("Docker Engine API", width,
			field("Host:", 9, m.host),
			field("Version:", 9, m.version),
		)
	}
	host := m.host
	if m.engine.Hostname != "" {
		host = m.engine.Hostname + " (" + m.host + ")"
	}
	return box("Docker Engine API", width,
		field("Host:", 9, host),
		field("Engine:", 9, engineLine(*m.engine)),
	)
}

// engineLine lists the most telling details first, so that the least
// important ones are cut off in narrow terminals.
func engineLine(e engine.Info) string {
	var parts []string
	if e.Version != "" {
		v := e.Version
		if e.APIVersion != "" {
			v += " (API " + e.APIVersion + ")"
		}
		parts = append(parts, v)
	}
	if e.OS != "" && e.Arch != "" {
		parts = append(parts, e.OS+"/"+e.Arch)
	}
	switch {
	case e.CPUs == 1:
		parts = append(parts, "1 CPU")
	case e.CPUs > 1:
		parts = append(parts, fmt.Sprintf("%d CPUs", e.CPUs))
	}
	if e.Memory > 0 {
		parts = append(parts, formatMemory(e.Memory))
	}
	return strings.Join(parts, " · ")
}

// formatMemory rounds to whole GiB, or MiB below 1 GiB.
func formatMemory(b int64) string {
	const mib, gib = 1 << 20, 1 << 30
	if b < gib {
		return fmt.Sprintf("%d MiB", (b+mib/2)/mib)
	}
	return fmt.Sprintf("%d GiB", (b+gib/2)/gib)
}

func (m Model) info(width int) string {
	if m.container != nil {
		return box("Info", width,
			field("Name:", 9, m.container.Name),
			field("Image:", 9, m.container.Image),
		)
	}
	// Unhealthy containers turn red so they're noticed when scrolled out of
	// view.
	unhealthy := theme.Value
	if m.unhealthy > 0 {
		unhealthy = theme.Alert
	}
	return box("Info", width,
		field("Running:", 9, fmt.Sprintf("%-6d", m.running))+
			field("Paused:", 11, fmt.Sprintf("%-6d", m.paused))+
			field("Exited:", 10, fmt.Sprint(m.exited)),
		field("Total:", 9, fmt.Sprintf("%-6d", m.total))+
			styledField("Unhealthy:", 11, fmt.Sprintf("%-6d", m.unhealthy), unhealthy)+
			field("Projects:", 10, fmt.Sprint(m.projects)),
	)
}

func field(key string, keyWidth int, value string) string {
	return styledField(key, keyWidth, value, theme.Value)
}

func styledField(key string, keyWidth int, value string, style lipgloss.Style) string {
	return theme.Key.Width(keyWidth).Render(key) + style.Render(value)
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
