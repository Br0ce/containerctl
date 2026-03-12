package view

import "github.com/rivo/tview"

type ErrorModal struct {
	name string
	*tview.Modal
}

func NewErrorModal() *ErrorModal {
	m := &ErrorModal{
		name:  "error",
		Modal: tview.NewModal(),
	}
	m.AddButtons([]string{"OK"})
	return m
}

func (m *ErrorModal) Populate(err error) {
	m.SetText("Error: " + err.Error())
}

func (m *ErrorModal) Name() string {
	return m.name
}
