package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Header struct {
	name string
	*tview.Flex
}

func NewHeader(dhost, dversion string) *Header {
	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("Docker Engine API: %s\nApi Version: %s", dhost, dversion)).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorAqua)

	appTitle := tview.NewTextView().
		SetText(`                 _        _                     _   _ 
  ___ ___  _ __ | |_ __ _(_)_ __   ___ _ __ ___| |_| |
 / __/ _ \| '_ \| __/ _` + "`" + ` | | '_ \ / _ \ '__/ __| __| |
| (_| (_) | | | | || (_| | | | | |  __/ | | (__| |_| |
 \___\___/|_| |_|\__\__,_|_|_| |_|\___|_|  \___|\__|_|`).
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorMidnightBlue)

	keyBindings := tview.NewTextView().
		SetText("<q>   Quit\n<l>   Logs\n<Esc> Back").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)

	return &Header{
		name: "header",
		Flex: tview.NewFlex().
			AddItem(contextView, 0, 1, false).
			AddItem(keyBindings, 0, 1, false).
			AddItem(appTitle, 0, 1, false),
	}
}

func (view *Header) Name() string {
	return view.name
}
