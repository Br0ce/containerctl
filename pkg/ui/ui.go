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

	header := view.NewHeader(dhost, cli.DaemonVersion())
	container := view.NewContainer()
	log := view.NewLog()
	errBar := view.NewErrorBar()

	pages := tview.NewPages().
		AddPage(container.Name(), container, true, true).
		AddPage(log.Name(), log, true, false)

	// Initial synchronous load before the app starts.
	if shorts, err := cli.Shorts(ctx); err != nil {
		errBar.Populate(err)
	} else {
		container.Populate(shorts)
	}

	app := tview.NewApplication().EnableMouse(true)
	// Start the background update loop.
	go update(ctx, app, pages, container, errBar, cli, updateRate)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 3, 0, false).
		AddItem(errBar, 1, 0, false).
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
				if name != container.Name() {
					return nil
				}
				row, _ := container.GetSelection()
				if row < 1 {
					// row 0 is the header — nothing to inspect.
					return nil
				}
				id := container.GetCell(row, 0).GetReference().(string)
				log.Populate(cli.Logs(ctx, id))
				pages.SwitchToPage(log.Name())
				app.SetFocus(log)
				return nil
			}

		case tcell.KeyEscape:
			// Return to the shorts table from any page.
			pages.SwitchToPage(container.Name())
			app.SetFocus(container)
			return nil
		}
		return event
	})

	return app.SetRoot(layout, true).Run()
}

func update(ctx context.Context, app *tview.Application, pages *tview.Pages, containerView *view.Container, statusBar *view.ErrorBar, cli *client.Client, rate time.Duration) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			app.QueueUpdateDraw(func() {
				// Only refresh while shorts is the active page.
				name, _ := pages.GetFrontPage()
				if name == containerView.Name() {
					if shorts, err := cli.Shorts(ctx); err != nil {
						statusBar.Populate(err)
					} else {
						statusBar.Clear()
						containerView.Populate(shorts)
					}
				}
			})
		case <-ctx.Done():
			return
		}
	}
}
