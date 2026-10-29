package containers

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	accent = lipgloss.Color("#1E90FF") // DodgerBlue

	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent)
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	cellStyle     = lipgloss.NewStyle().Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Background(accent).Foreground(lipgloss.Color("#FFFFFF"))
)

// stateCell colors a container state. It resets only the foreground color
// (SGR 39) instead of all attributes, so the background of the selected row
// is kept for the rest of the row.
func stateCell(state string) string {
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
