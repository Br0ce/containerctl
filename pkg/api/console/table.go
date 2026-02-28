package console

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/cctl/pkg/client"
	"github.com/Br0ce/cctl/pkg/container"
)

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

func PopulateShortsTable(table *tview.Table, shorts []container.Short) {
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

func PopulateLogsView(view *tview.TextView, logs client.LogSeq) {
	var sb strings.Builder
	for line, err := range logs {
		if err != nil {
			sb.WriteString("[red]error: " + err.Error() + "[-]\n")
			break
		}
		sb.WriteString(tview.TranslateANSI(line) + "\n")
	}
	view.SetText(sb.String())
}
