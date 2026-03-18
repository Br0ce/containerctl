package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/container"
)

type Container struct {
	name string
	*tview.Table
	infoHeader []InfoItem
}

func NewContainer() *Container {
	shortsView := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	shortsView.SetBorder(true).SetTitle(" Containers ").SetTitleColor(tcell.ColorDodgerBlue)
	shortsView.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDodgerBlue).Foreground(tcell.ColorWhite))
	return &Container{
		name:  "container",
		Table: shortsView,
	}
}

func (view *Container) Populate(shorts []container.Short) {
	view.Clear()
	view.ScrollToBeginning()

	for col, h := range []string{"ID", "Name", "Image", "Status", "State"} {
		view.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1))
	}

	for row, short := range shorts {
		color := color(short.State)
		view.SetCell(row+1, 0, tview.NewTableCell(short.ID[:12]).SetTextColor(color).SetAlign(tview.AlignLeft).SetExpansion(1).SetReference(short.ID))
		view.SetCell(row+1, 1, tview.NewTableCell(short.Name).SetTextColor(color).SetAlign(tview.AlignLeft).SetMaxWidth(20))
		view.SetCell(row+1, 2, tview.NewTableCell(short.Image).SetTextColor(color).SetAlign(tview.AlignLeft).SetMaxWidth(20))
		view.SetCell(row+1, 3, tview.NewTableCell(short.Status).SetTextColor(color).SetAlign(tview.AlignLeft).SetMaxWidth(20))
		view.SetCell(row+1, 4, tview.NewTableCell(short.State).SetTextColor(color).SetAlign(tview.AlignLeft).SetExpansion(1))
	}

	view.infoHeader = containerSummary(shorts)
}

func (view *Container) Name() string {
	return view.name
}

func (view *Container) KeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: tcell.KeyRune, Rune: 'q', Desc: "Quit"},
		{Key: tcell.KeyRune, Rune: 'l', Desc: "Logs"},
		{Key: tcell.KeyRune, Rune: 'f', Desc: "Files"},
		{Key: tcell.KeyRune, Rune: 's', Desc: "Start"},
		{Key: tcell.KeyRune, Rune: 'x', Desc: "Stop"},
		{Key: tcell.KeyRune, Rune: 'p', Desc: "Pause"},
		{Key: tcell.KeyRune, Rune: 'u', Desc: "Unpause"},
	}
}

func color(state string) tcell.Color {
	switch state {
	case "running":
		return tcell.ColorGreen
	case "paused":
		return tcell.ColorGold
	case "exited":
		return tcell.ColorDimGray
	case "removing", "dead":
		return tcell.ColorOrangeRed
	case "created", "restarting":
		return tcell.ColorSteelBlue
	default:
		return tcell.ColorDimGray
	}
}

func (view *Container) InfoHeader() []InfoItem {
	return view.infoHeader
}

func containerSummary(shorts []container.Short) []InfoItem {
	var running, paused int
	for _, s := range shorts {
		switch s.State {
		case "running":
			running++
		case "paused":
			paused++
		default:
		}
	}
	return []InfoItem{
		{Key: "Running", Value: fmt.Sprintf("%d", running)},
		{Key: "Stopped", Value: fmt.Sprintf("%d", len(shorts))},
		{Key: "Paused", Value: fmt.Sprintf("%d", paused)},
	}
}
