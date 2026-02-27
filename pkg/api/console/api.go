package console

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/cctl/pkg/client"
)

func Run(ctx context.Context) error {
	cli, err := client.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	shorts, err := cli.All(ctx)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	app := tview.NewApplication()

	// Header bar: key bindings on the left, app title on the right.
	keyBindings := tview.NewTextView().
		SetText("<q> Quit").
		SetTextColor(tcell.ColorYellow)

	appTitle := tview.NewTextView().
		SetText("cctl").
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorWhite)

	header := tview.NewFlex().
		AddItem(keyBindings, 0, 1, false).
		AddItem(appTitle, 0, 1, false)

	// Container table with border frame.
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.SetBorder(true).SetTitle(" Containers ")

	headers := []string{"ID", "Name", "Image", "Status"}
	for col, h := range headers {
		table.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1))
	}

	for row, short := range shorts {
		table.SetCell(row+1, 0, tview.NewTableCell(short.ID[:12]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell(short.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell(short.Image).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell(short.Status).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(1))
	}

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(table, 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}
