package view

import "github.com/rivo/tview"

const ErrorBar = "errorBar"

func NewErrorBar() *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
}

func PopulateErrorBar(view *tview.TextView, err error) {
	view.SetText("[red]Error: " + err.Error() + "[-]")
}
