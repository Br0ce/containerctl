package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/view"
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

	dhost, err := cli.DaemonHostname()
	if err != nil {
		return fmt.Errorf("get daemon hostname: %w", err)
	}

	headerView := view.NewHeader(dhost, cli.DaemonVersion())
	containerView := view.NewContainer()
	logView := view.NewLog()
	errView := view.NewErrorBar()

	pages := tview.NewPages().
		AddPage(view.Container, containerView, true, true).
		AddPage(view.Log, logView, true, false)

	// Initial synchronous load before the app starts.
	if shorts, err := cli.Shorts(ctx); err != nil {
		view.PopulateErrorBar(errView, err)
	} else {
		view.PopulateContainer(containerView, shorts)
	}

	app := tview.NewApplication().EnableMouse(true)
	// Start the background update loop.
	go update(ctx, app, pages, containerView, errView, cli, updateRate)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerView, 3, 0, false).
		AddItem(errView, 1, 0, false).
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
				if name != view.Container {
					return nil
				}
				row, _ := containerView.GetSelection()
				if row < 1 {
					// row 0 is the header — nothing to inspect.
					return nil
				}
				id := containerView.GetCell(row, 0).GetReference().(string)
				view.PopulateLog(logView, cli.Logs(ctx, id))
				pages.SwitchToPage(view.Log)
				app.SetFocus(logView)
				return nil
			}

		case tcell.KeyEscape:
			// Return to the shorts table from any page.
			pages.SwitchToPage(view.Container)
			app.SetFocus(containerView)
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}

func update(ctx context.Context, app *tview.Application, pages *tview.Pages, containersView *tview.Table, statusBar *tview.TextView, cli *client.Client, rate time.Duration) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			app.QueueUpdateDraw(func() {
				// Only refresh while shorts is the active page.
				name, _ := pages.GetFrontPage()
				if name == view.Container {
					if shorts, err := cli.Shorts(ctx); err != nil {
						statusBar.SetText("[red]Error: " + err.Error() + "[-]")
					} else {
						statusBar.Clear()
						view.PopulateContainer(containersView, shorts)
					}
				}
			})
		case <-ctx.Done():
			return
		}
	}
}
