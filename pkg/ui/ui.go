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

type UI struct {
	app       *tview.Application
	cli       *client.Client
	container *view.Container
	errBar    *view.ErrorBar
	header    *view.Header
	body      *tview.Pages
	log       *view.Log
}

// New initializes the UI components and returns a UI instance.
// It does not start the TUI application; the caller must call Run() to do so.
// Note that the caller is responsible for calling Close() on the UI to clean up
// resources after Run() returns.
func New(host string) (*UI, error) {
	cli, err := client.New(host)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	header := view.NewHeader(cli.DaemonHostname(), cli.DaemonVersion())
	container := view.NewContainer()
	log := view.NewLog()
	errBar := view.NewErrorBar()

	body := tview.NewPages().
		AddPage(container.Name(), container, true, true).
		AddPage(log.Name(), log, true, false)

	return &UI{
		app:       tview.NewApplication().EnableMouse(true),
		cli:       cli,
		header:    header,
		container: container,
		log:       log,
		errBar:    errBar,
		body:      body,
	}, nil
}

// Run starts the TUI application and blocks until it exits.
func (ui *UI) Run(ctx context.Context) error {
	// Set up a cancellable context for input capture to cancel the update loop.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go ui.updateLoop(ctx, updateRate)
	ui.inputCapture(ctx, cancel)

	// Initial synchronous load before the app starts.
	if shorts, err := ui.cli.Shorts(ctx); err != nil {
		ui.errBar.Populate(err)
	} else {
		ui.container.Populate(shorts)
	}

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.header, 3, 0, false).
		AddItem(ui.errBar, 1, 0, false).
		AddItem(ui.body, 0, 1, true)

	return ui.app.SetRoot(root, true).Run()
}

func (ui *UI) Close() error {
	return ui.cli.Close()
}

// updateLoop periodically refreshes the app contents. The loop is cancellable via the provided context.
func (ui *UI) updateLoop(ctx context.Context, rate time.Duration) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ui.app.QueueUpdateDraw(func() {
				// Only refresh while containers is the active page.
				name, _ := ui.body.GetFrontPage()
				if name == ui.container.Name() {
					if shorts, err := ui.cli.Shorts(ctx); err != nil {
						ui.errBar.Populate(err)
					} else {
						ui.errBar.Clear()
						ui.container.Populate(shorts)
					}
				}
			})
		case <-ctx.Done():
			return
		}
	}
}

// inputCapture sets up global keybindings for the app.
func (ui *UI) inputCapture(ctx context.Context, cancel context.CancelFunc) {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				cancel()
				ui.app.Stop()
				return nil

			case 'l':
				// Only act when the table is the containers page.
				name, _ := ui.body.GetFrontPage()
				if name != ui.container.Name() {
					return nil
				}
				row, _ := ui.container.GetSelection()
				if row < 1 {
					// row 0 is the header — nothing to inspect.
					return nil
				}
				id := ui.container.GetCell(row, 0).GetReference().(string)
				ui.log.Populate(ui.cli.Logs(ctx, id))
				ui.body.SwitchToPage(ui.log.Name())
				ui.app.SetFocus(ui.log)
				return nil
			}

		case tcell.KeyEscape:
			// Return to the default page from any page.
			ui.body.SwitchToPage(ui.container.Name())
			ui.app.SetFocus(ui.container)
			return nil
		}
		return event
	})
}
