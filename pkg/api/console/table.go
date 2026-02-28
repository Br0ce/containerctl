package console

import (
	"context"
	"strings"

	"github.com/Br0ce/cctl/pkg/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func PopulateShortsTable(ctx context.Context, table *tview.Table, cli *client.Client) {
	shorts, err := cli.Shorts(ctx)
	if err != nil {
		// return fmt.Errorf("list short description of containers: %w", err)
	}
	table.Clear()

	for col, h := range []string{"ID", "Name", "Image", "Status"} {
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
	}
}

func PopulateLogsView(view *tview.TextView, logs client.LogSeq) {
	var sb strings.Builder
	for line, err := range logs {
		if err != nil {
			sb.WriteString("[red]error: " + err.Error() + "[-]\n")
			break
		}
		sb.WriteString(line + "\n")
	}
	view.SetText(sb.String())
}
