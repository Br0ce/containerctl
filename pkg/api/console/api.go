package console

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/cctl/pkg/client"
)

const updateRate = 3 * time.Second

func Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cli, err := client.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer cli.Close()

	app := tview.NewApplication().EnableMouse(true)

	keyBindings := tview.NewTextView().
		SetText("<q>   Quit\n<l>   Logs\n<Esc> Back").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)

	dhost, err := cli.DaemonHostname()
	if err != nil {
		return fmt.Errorf("get daemon hostname: %w", err)
	}

	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("Daemon Host: %s\nApi Version: %s", dhost, cli.DaemonVersion())).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorAqua)

	appTitle := tview.NewTextView().
		SetText("cctl").
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorWhite)

	header := tview.NewFlex().
		AddItem(contextView, 0, 1, false).
		AddItem(keyBindings, 0, 1, false).
		AddItem(appTitle, 0, 1, false)

	containersView := CreateContainersView()
	logsView := CreateLogsView()

	pages := tview.NewPages().
		AddPage(containersPage, containersView, true, true).
		AddPage(logsPage, logsView, true, false)

	// Initial synchronous load before the app starts.
	if shorts, err := cli.Shorts(ctx); err == nil {
		PopulateContainersView(containersView, shorts)
	}

	// Start the background update loop.
	go update(ctx, app, pages, containersView, cli, updateRate)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 3, 0, false).
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
				// Only act when the table is the shorts page.
				name, _ := pages.GetFrontPage()
				if name != containersPage {
					return nil
				}
				row, _ := containersView.GetSelection()
				if row < 1 {
					// row 0 is the header — nothing to inspect.
					return nil
				}
				id := containersView.GetCell(row, 0).GetReference().(string)
				PopulateLogsView(logsView, cli.Logs(ctx, id))
				pages.SwitchToPage(logsPage)
				app.SetFocus(logsView)
				return nil
			}

		case tcell.KeyEscape:
			// Return to the shorts table from any page.
			pages.SwitchToPage(containersPage)
			app.SetFocus(containersView)
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}

func update(ctx context.Context, app *tview.Application, pages *tview.Pages, containersView *tview.Table, cli *client.Client, rate time.Duration) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			app.QueueUpdateDraw(func() {
				// Only refresh while shorts is the active page.
				name, _ := pages.GetFrontPage()
				if name == containersPage {
					if shorts, err := cli.Shorts(ctx); err == nil {
						PopulateContainersView(containersView, shorts)
					}
				}
			})
		case <-ctx.Done():
			return
		}
	}
}
