package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/container"
)

const Container = "containers"

func NewContainer() *tview.Table {
	shortsView := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	shortsView.SetBorder(true).SetTitle(" Containers ")
	return shortsView
}

func PopulateContainer(table *tview.Table, shorts []container.Short) {
	table.Clear()

	for col, h := range []string{"ID", "Name", "Image", "Status", "State"} {
		table.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1))
	}

	for row, short := range shorts {
		table.SetCell(row+1, 0, tview.NewTableCell(short.ID[:12]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1).SetReference(short.ID))
		table.SetCell(row+1, 1, tview.NewTableCell(short.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell(short.Image).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell(short.Status).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 4, tview.NewTableCell(stateDot(short.State)).SetAlign(tview.AlignCenter).SetExpansion(1))
	}
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
