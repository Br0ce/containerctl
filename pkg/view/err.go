package view

import "github.com/rivo/tview"

type ErrorBar struct {
	name string
	*tview.TextView
}

func NewErrorBar() *ErrorBar {
	return &ErrorBar{
		name: "errorbar",
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignLeft),
	}
}

func (view *ErrorBar) Populate(err error) {
	view.SetText("[red]Error: " + err.Error() + "[-]")
}

func (view *ErrorBar) Name() string {
	return view.name
}
