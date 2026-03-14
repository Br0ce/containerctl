package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Header struct {
	name        string
	keyBindings *tview.TextView
	*tview.Flex
}

func NewHeader(dhost, dversion string) *Header {
	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("Docker Engine API Host:    [white]%s[aqua]\nDocker Engine API Version: [white]%s", dhost, dversion)).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorAqua).
		SetDynamicColors(true)

	appTitle := tview.NewTextView().
		SetText(`                 _        _                     _   _ 
  ___ ___  _ __ | |_ __ _(_)_ __   ___ _ __ ___| |_| |
 / __/ _ \| '_ \| __/ _` + "`" + ` | | '_ \ / _ \ '__/ __| __| |
| (_| (_) | | | | || (_| | | | | |  __/ | | (__| |_| |
 \___\___/|_| |_|\__\__,_|_|_| |_|\___|_|  \___|\__|_|`).
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorMediumBlue)

	keyBindings := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	h := &Header{
		name:        "header",
		keyBindings: keyBindings,
		Flex: tview.NewFlex().
			AddItem(contextView, 0, 1, false).
			AddItem(keyBindings, 0, 1, false).
			AddItem(appTitle, 0, 1, false),
	}
	h.SetKeyBindings("")

	return h
}

// SetKeyBindings updates the key bindings display for the given page name.
func (h *Header) SetKeyBindings(name string) {
	switch name {
	case "log":
		h.keyBindings.SetText("[dodgerblue]<q>[gray]   Quit\n[dodgerblue]<Esc>[gray] Back")
	case "files":
		h.keyBindings.SetText("[dodgerblue]<q>[gray]   Quit\n[dodgerblue]<Esc>[gray] Back\n[dodgerblue]<Enter>[gray] Open / Parent dir")
	default:
		h.keyBindings.SetText(`[dodgerblue]<q>[gray]   Quit      [dodgerblue]<u>[gray]   Unpause
[dodgerblue]<l>[gray]   Logs      [dodgerblue]<p>[gray]   Pause
[dodgerblue]<s>[gray]   Start     [dodgerblue]<f>[gray]   Files
[dodgerblue]<x>[gray]   Stop`)
	}
}

func (view *Header) Name() string {
	return view.name
}
