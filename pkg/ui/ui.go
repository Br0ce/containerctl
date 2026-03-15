package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/file"
	"github.com/Br0ce/containerctl/pkg/view"
)

const updateRate = 3 * time.Second

type Config struct {
	Host         string
	DockerHost   string
	Username     string
	IdentityFile string
	AskPassword  bool
}

type UI struct {
	app       *tview.Application
	cli       *client.Client
	container *view.Container
	errModal  *view.ErrorModal
	header    *view.Header
	body      *tview.Pages
	rootPages *tview.Pages
	errCh     chan error
	log       *view.Log
	files     *view.Files
}

// New initializes the UI components and returns a UI instance.
// It does not start the TUI application; the caller must call Run() to do so.
// Note that the caller is responsible for calling Close() on the UI to clean up
// resources after Run() returns.
func New(cfg Config) (*UI, error) {
	var opts []client.ClientOptions
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.DockerHost == "" {
		return nil, fmt.Errorf("docker host is required")
	}
	opts = append(opts, client.WithHost(cfg.Host), client.WithDockerHost(cfg.DockerHost))

	if cfg.Username != "" {
		opts = append(opts, client.WithUsername(cfg.Username))
	}
	if cfg.IdentityFile != "" {
		opts = append(opts, client.WithIdentityFile(cfg.IdentityFile))
	}
	if cfg.AskPassword {
		opts = append(opts, client.WithAskPassword(true))
	}

	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	app := tview.NewApplication().EnableMouse(true)
	header := view.NewHeader(cli.DaemonHost(), cli.DaemonVersion())
	container := view.NewContainer()
	log := view.NewLog()
	files := view.NewFiles()
	errModal := view.NewErrorModal()

	body := tview.NewPages().
		AddPage(container.Name(), container, true, true).
		AddPage(log.Name(), log, true, false).
		AddPage(files.Name(), files, true, false)

	ui := &UI{
		app:       app,
		cli:       cli,
		header:    header,
		container: container,
		log:       log,
		files:     files,
		errModal:  errModal,
		// Buffered channel to avoid blocking when publishing errors. Needs rework.
		errCh: make(chan error, 1),
		body:  body,
	}
	header.SetKeyBindings(container.KeyBindings())
	return ui, nil
}

// Run starts the TUI application and blocks until it exits.
func (ui *UI) Run(ctx context.Context) error {
	// Set up a cancellable context for input capture to cancel the update loop.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.header, 5, 0, false).
		AddItem(ui.body, 0, 1, true)

	ui.rootPages = tview.NewPages().
		AddPage("main", mainLayout, true, true).
		AddPage(ui.errModal.Name(), ui.errModal, true, false)

	ui.errModal.SetDoneFunc(func(_ int, _ string) {
		ui.rootPages.HidePage(ui.errModal.Name())
		ui.app.SetFocus(ui.body)
	})

	go ui.updateLoop(ctx, updateRate)
	go ui.listenErrors(ctx)
	ui.inputCapture(ctx, cancel)

	// Initial load: run async so the modal can be shown if an error occurs.
	go func() {
		shorts, err := ui.cli.AllShorts(ctx)
		if err != nil {
			ui.publishError(err)
			return
		}

		ui.app.QueueUpdateDraw(func() { ui.container.Populate(shorts) })
	}()

	return ui.app.SetRoot(ui.rootPages, true).Run()
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
					ui.populateContainers(ctx)
				}
			})
		case <-ctx.Done():
			return
		}
	}
}

func (ui *UI) populateContainers(ctx context.Context) {
	shorts, err := ui.cli.AllShorts(ctx)
	if err != nil {
		ui.publishError(err)
		return
	}
	ui.container.Populate(shorts)
}

// inputCapture sets up keybindings for the app.
// View-specific bindings are dispatched first, then global bindings (q, Esc).
func (ui *UI) inputCapture(ctx context.Context, cancel context.CancelFunc) {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// View-specific keys.
		name, _ := ui.body.GetFrontPage()
		switch name {
		case ui.container.Name():
			if event.Key() == tcell.KeyRune {
				switch event.Rune() {
				case 'l':
					ui.handleLogs(ctx)
					return nil
				case 's':
					ui.handleStart(ctx)
					return nil
				case 'x':
					ui.handleStop(ctx)
					return nil
				case 'p':
					ui.handlePause(ctx)
					return nil
				case 'u':
					ui.handleUnpause(ctx)
					return nil
				case 'f':
					ui.handleFiles(ctx)
					return nil
				}
			}
		case ui.files.Name():
			if event.Key() == tcell.KeyEnter {
				ui.handleFiles(ctx)
				return nil
			}
		}

		// Global keys.
		switch event.Key() {
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				cancel()
				ui.app.Stop()
				return nil
			}
		case tcell.KeyEscape:
			// If the error modal is front, let it handle Esc via its done func.
			frontName, _ := ui.rootPages.GetFrontPage()
			if frontName == ui.errModal.Name() {
				return event
			}
			// Return to the container page from any view.
			ui.body.SwitchToPage(ui.container.Name())
			ui.header.SetKeyBindings(ui.container.KeyBindings())
			ui.app.SetFocus(ui.container)
			return nil
		}
		return event
	})
}

func (ui *UI) handleLogs(ctx context.Context) {
	// Only act when the table is the containers page.
	name, _ := ui.body.GetFrontPage()
	if name != ui.container.Name() {
		return
	}
	row, _ := ui.container.GetSelection()
	if row < 1 {
		// row 0 is the header — nothing to inspect.
		return
	}
	id := ui.container.GetCell(row, 0).GetReference().(string)
	ui.log.Populate(ui.cli.Logs(ctx, id))
	ui.body.SwitchToPage(ui.log.Name())
	ui.header.SetKeyBindings(ui.log.KeyBindings())
	ui.app.SetFocus(ui.log)
}

func (ui *UI) handleFiles(ctx context.Context) {
	name, _ := ui.body.GetFrontPage()

	switch name {
	case ui.container.Name():
		row, _ := ui.container.GetSelection()
		if row < 1 {
			return
		}
		id := ui.container.GetCell(row, 0).GetReference().(string)
		files, err := ui.cli.FilesIn(ctx, file.Info{ContainerID: id})
		if err != nil {
			ui.publishError(err)
			return
		}
		ui.files.Populate(files)
		ui.body.SwitchToPage(ui.files.Name())
		ui.header.SetKeyBindings(ui.files.KeyBindings())
		ui.app.SetFocus(ui.files)
	case ui.files.Name():
		row, _ := ui.files.GetSelection()
		if row < 0 {
			return
		}

		selection, ok := ui.files.GetCell(row, 0).GetReference().(file.Info)
		if !ok {
			ui.publishError(fmt.Errorf("invalid file reference"))
			return
		}

		// If the selection is not a directory, do nothing on Enter.
		if !selection.IsDir {
			return
		}

		files, err := ui.cli.FilesIn(ctx, selection)
		if err != nil {
			ui.publishError(err)
			return
		}
		ui.files.Populate(files)
		ui.body.SwitchToPage(ui.files.Name())
		ui.header.SetKeyBindings(ui.files.KeyBindings())
		ui.app.SetFocus(ui.files)
	}
}

func (ui *UI) handleStart(ctx context.Context) {
	name, _ := ui.body.GetFrontPage()
	if name != ui.container.Name() {
		return
	}
	row, _ := ui.container.GetSelection()
	if row < 1 {
		return
	}
	id := ui.container.GetCell(row, 0).GetReference().(string)

	// Start the container in the background.
	go func() {
		err := ui.cli.StartContainer(ctx, id)
		if err != nil {
			ui.publishError(err)
		}
	}()
	ui.populateContainers(ctx)
}

func (ui *UI) handleStop(ctx context.Context) {
	name, _ := ui.body.GetFrontPage()
	if name != ui.container.Name() {
		return
	}
	row, _ := ui.container.GetSelection()
	if row < 1 {
		return
	}
	id := ui.container.GetCell(row, 0).GetReference().(string)

	// Stop the container in the background.
	go func() {
		err := ui.cli.StopContainer(ctx, id)
		if err != nil {
			ui.publishError(err)
		}
	}()
	ui.populateContainers(ctx)
}

func (ui *UI) handlePause(ctx context.Context) {
	name, _ := ui.body.GetFrontPage()
	if name != ui.container.Name() {
		return
	}
	row, _ := ui.container.GetSelection()
	if row < 1 {
		return
	}
	id := ui.container.GetCell(row, 0).GetReference().(string)

	// Pause the container in the background.
	go func() {
		err := ui.cli.PauseContainer(ctx, id)
		if err != nil {
			ui.publishError(err)
		}
	}()
	ui.populateContainers(ctx)
}

func (ui *UI) handleUnpause(ctx context.Context) {
	name, _ := ui.body.GetFrontPage()
	if name != ui.container.Name() {
		return
	}
	row, _ := ui.container.GetSelection()
	if row < 1 {
		return
	}
	id := ui.container.GetCell(row, 0).GetReference().(string)

	// Unpause the container in the background.
	go func() {
		err := ui.cli.UnpauseContainer(ctx, id)
		if err != nil {
			ui.publishError(err)
		}
	}()
	ui.populateContainers(ctx)
}

// publishError sends err to errCh. If the channel is full, the error is dropped.
func (ui *UI) publishError(err error) {
	select {
	case ui.errCh <- err:
	default:
	}
}

// listenErrors reads from errCh and displays the error modal.
func (ui *UI) listenErrors(ctx context.Context) {
	for {
		select {
		case err := <-ui.errCh:
			ui.app.QueueUpdateDraw(func() {
				name, _ := ui.rootPages.GetFrontPage()
				if name == ui.errModal.Name() {
					return
				}
				ui.errModal.Populate(err)
				ui.rootPages.ShowPage(ui.errModal.Name())
				ui.app.SetFocus(ui.errModal)
			})
		case <-ctx.Done():
			return
		}
	}
}
