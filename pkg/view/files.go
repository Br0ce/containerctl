package view

import (
	"fmt"

	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/file"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Files struct {
	name string
	*tview.Table
	curShort *container.Short
}

func NewFiles() *Files {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.SetBorder(true).SetTitleColor(tcell.ColorDodgerBlue)

	return &Files{
		name:  "files",
		Table: table,
	}
}

func (view *Files) Populate(files []file.Info, short container.Short) {
	view.Clear()
	view.ScrollToBeginning()

	if len(files) == 0 {
		return
	}

	view.SetTitle(fmt.Sprintf(" %s ", files[0].DisplayName))
	for row, file := range files {
		color := tcell.ColorDimGray
		if file.IsDir {
			color = tcell.ColorDodgerBlue
		}
		view.SetCell(row, 0,
			tview.NewTableCell(file.Name).
				SetTextColor(color).
				SetExpansion(1).
				SetReference(file))
	}
	view.curShort = &short
}

func (view *Files) Name() string {
	return view.name
}

func (view *Files) KeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: tcell.KeyRune, Rune: 'q', Desc: "Quit"},
		{Key: tcell.KeyEsc, Desc: "Back"},
		{Key: tcell.KeyEnter, Desc: "Open"},
	}
}

func (view *Files) InfoHeader() []InfoItem {
	if view.curShort == nil {
		return nil
	}
	return []InfoItem{
		{Key: "Name", Value: view.curShort.Name},
		{Key: "Image", Value: view.curShort.Image},
	}
}
