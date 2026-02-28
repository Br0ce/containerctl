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
		SetText("<q> Quit  <l> Inspect  <Esc> Back").
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

	// Metadata view shown when the user presses "l" on a row.
	metaView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	metaView.SetBorder(true).SetTitle(" Container Metadata ")

	// Pages holds both views and exposes one at a time.
	// "table" is the default visible page; "meta" starts hidden.
	pages := tview.NewPages().
		AddPage("table", table, true, true).
		AddPage("meta", metaView, true, false)

	// Initial synchronous load before the app starts.
	PopulateShortsTable(ctx, table, cli)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					// Only refresh while the table is the active page.
					name, _ := pages.GetFrontPage()
					if name == "table" {
						PopulateShortsTable(ctx, table, cli)
					}
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(pages, 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				cancel()
				app.Stop()
				return nil

			case 'l':
				// Only act when the table is the front page.
				name, _ := pages.GetFrontPage()
				if name != "table" {
					return nil
				}
				row, _ := table.GetSelection()
				if row < 1 {
					// row 0 is the header — nothing to inspect.
					return nil
				}
				id := table.GetCell(row, 0).GetReference().(string)
				PopulateLogsView(metaView, cli.Logs(ctx, id))
				pages.SwitchToPage("meta")
				app.SetFocus(metaView)
				return nil
			}

		case tcell.KeyEscape:
			// Return to the container table from any page.
			pages.SwitchToPage("table")
			app.SetFocus(table)
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}
