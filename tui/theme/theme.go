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

	Title  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	Key    = lipgloss.NewStyle().Foreground(accent)
	Value  = lipgloss.NewStyle().Foreground(text)
	Header = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	Cell   = lipgloss.NewStyle().Padding(0, 1)
	// Selected keeps the terminal's default foreground: colored cells reset
	// to it with SGR 39, so any other foreground would break off after them.
	Selected = lipgloss.NewStyle().Background(accent)

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

// StateCell renders a container state as a dot in the state's color. It
// resets only the foreground color (SGR 39) instead of all attributes, so the
// background of the selected row is kept for the rest of the row.
func StateCell(state string) string {
	return ansi.Style{}.ForegroundColor(stateColor(state)).String() + "●" +
		ansi.Style{}.ForegroundColor(nil).String()
}

// DirCell marks a directory name in bold. Like StateCell, it resets only the
// bold attribute (SGR 22), so the selected row keeps its colors; a colored
// name would vanish on the accent background.
func DirCell(name string) string {
	return ansi.Style{}.Bold().String() + name + ansi.Style{}.Normal().String()
}

// Dim renders s faint. Faint is used instead of a color for the same reason
// as in DirCell, and resets the same way.
func Dim(s string) string {
	return ansi.Style{}.Faint().String() + s + ansi.Style{}.Normal().String()
}

// FileCell dims a file name so that directories stand out.
func FileCell(name string) string {
	return Dim(name)
}

func stateColor(state string) color.Color {
	switch state {
	case "running":
		return lipgloss.Color("#4ADE80") // Bright green
	case "paused":
		return lipgloss.Color("#FACC15") // Yellow
	case "removing", "dead":
		return lipgloss.Color("#F87171") // Bright red
	case "created", "restarting":
		return lipgloss.Color("#C084FC") // Violet; blue would vanish on the selection bar
	default:
		return lipgloss.Color("#9CA3AF") // Light gray
	}
}
