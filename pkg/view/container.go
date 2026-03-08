package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/container"
)

type Container struct {
	name string
	*tview.Table
}

func NewContainer() *Container {
	shortsView := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	shortsView.SetBorder(true).SetTitle(" Containers ").SetTitleColor(tcell.ColorAqua)
	return &Container{
		name:  "container",
		Table: shortsView,
	}
}

func (view *Container) Populate(shorts []container.Short) {
	view.Clear()

	for col, h := range []string{"ID", "Name", "Image", "Status", "State"} {
		view.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1))
	}

	for row, short := range shorts {
		view.SetCell(row+1, 0, tview.NewTableCell(short.ID[:12]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1).SetReference(short.ID))
		view.SetCell(row+1, 1, tview.NewTableCell(short.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		view.SetCell(row+1, 2, tview.NewTableCell(short.Image).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		view.SetCell(row+1, 3, tview.NewTableCell(short.Status).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		view.SetCell(row+1, 4, tview.NewTableCell(stateDot(short.State)).SetAlign(tview.AlignCenter).SetExpansion(1))
	}
}

func (view *Container) Name() string {
	return view.name
}

func stateDot(s container.State) string {
	switch s {
	case container.Green:
		return "[green]●[-]"
	case container.Red:
		return "[red]●[-]"
	default:
		return "[yellow]●[-]"
	}
}
