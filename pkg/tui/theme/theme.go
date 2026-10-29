// Package theme holds the colors and styles shared by the TUI panes.
package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	accent = lipgloss.Color("#1E90FF") // DodgerBlue
	text   = lipgloss.Color("#C0C0C0") // Silver
	white  = lipgloss.Color("#FFFFFF")

	Title    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	Key      = lipgloss.NewStyle().Foreground(accent)
	Value    = lipgloss.NewStyle().Foreground(text)
	Header   = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	Cell     = lipgloss.NewStyle().Padding(0, 1)
	Selected = lipgloss.NewStyle().Background(accent).Foreground(white)

	box = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent)
)

// BoxFrame is the number of columns and rows a Box adds around its content.
const (
	BoxFrameWidth  = 2
	BoxFrameHeight = 2
)

// Box renders content in a rounded border of the given outer width with
// title centered in the top border. Lipgloss borders have no titles, so the
// title is composed as a layer on top of the border. Content lines that don't
// fit are truncated, not wrapped.
func Box(title string, width int, content string) string {
	inner := max(width-BoxFrameWidth, 0)
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, inner, "")
	}
	body := box.Width(width).Render(strings.Join(lines, "\n"))

	label := Title.Render(ansi.Truncate(" "+title+" ", inner, ""))
	x := (width - lipgloss.Width(label)) / 2
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(body),
		lipgloss.NewLayer(label).X(x),
	).Render()
}

// StateCell colors a container state. It resets only the foreground color
// (SGR 39) instead of all attributes, so the background of the selected row
// is kept for the rest of the row.
func StateCell(state string) string {
	return ansi.Style{}.ForegroundColor(stateColor(state)).String() + state +
		ansi.Style{}.ForegroundColor(nil).String()
}

func stateColor(state string) color.Color {
	switch state {
	case "running":
		return lipgloss.Color("#008000") // Green
	case "paused":
		return lipgloss.Color("#FFD700") // Gold
	case "removing", "dead":
		return lipgloss.Color("#FF4500") // OrangeRed
	case "created", "restarting":
		return lipgloss.Color("#4682B4") // SteelBlue
	default:
		return lipgloss.Color("#696969") // DimGray
	}
}
