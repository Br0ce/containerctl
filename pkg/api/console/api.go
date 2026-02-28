package console

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/cctl/pkg/client"
)

func Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cli, err := client.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
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

	// Initial synchronous load before the app starts.
	PopulateShortsTable(ctx, table, cli)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					PopulateShortsTable(ctx, table, cli)
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(table, 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			cancel()
			app.Stop()
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}
