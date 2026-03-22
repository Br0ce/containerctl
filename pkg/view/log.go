package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/container"
)

type Log struct {
	name string
	*tview.TextView
	curShort *container.Short
}

func NewLog() *Log {
	logsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	logsView.SetBorder(true).SetTitle(" Logs ").SetTitleColor(tcell.ColorDodgerBlue)
	return &Log{
		name:     "log",
		TextView: logsView,
	}
}

// InitSession clears the log view and sets the container context for InfoHeader.
// Must be called from the main goroutine before streaming begins.
func (view *Log) InitSession(short container.Short) {
	view.curShort = &short
	view.Clear()
}

// Populate appends a single log line to the view. To initiate a log session,
// call InitSession first to set the container context and clear the text.
func (view *Log) Populate(line string) {
	fmt.Fprintf(view, "%s\n", tview.TranslateANSI(line))
	view.ScrollToEnd()
}

func (view *Log) Name() string {
	return view.name
}

func (view *Log) KeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: tcell.KeyRune, Rune: 'q', Desc: "Quit"},
		{Key: tcell.KeyEsc, Desc: "Back"},
	}
}

func (view *Log) InfoHeader() []InfoItem {
	if view.curShort == nil {
		return nil
	}
	return []InfoItem{
		{Key: "Name", Value: view.curShort.Name},
		{Key: "Image", Value: view.curShort.Image},
	}
}
