package view

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewHeader(dhost, dversion string) *tview.Flex {
	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("Daemon Host: %s\nApi Version: %s", dhost, dversion)).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorAqua)

	appTitle := tview.NewTextView().
		SetText("cctl").
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorWhite)

	keyBindings := tview.NewTextView().
		SetText("<q>   Quit\n<l>   Logs\n<Esc> Back").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)

	return tview.NewFlex().
		AddItem(contextView, 0, 1, false).
		AddItem(keyBindings, 0, 1, false).
		AddItem(appTitle, 0, 1, false)
}
