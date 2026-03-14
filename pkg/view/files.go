package view

import (
	"io/fs"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Files struct {
	name string
	*tview.Table
}

func NewFiles() *Files {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.SetBorder(true).SetTitleColor(tcell.ColorAqua)

	return &Files{
		name:  "files",
		Table: table,
	}
}

func (view *Files) Populate(files []fs.FileInfo) {
	view.Clear()

	for row, file := range files {
		color := tcell.ColorGray
		if file.IsDir() {
			color = tcell.ColorAqua
		}
		view.SetCell(row, 0,
			tview.NewTableCell(file.Name()).
				SetTextColor(color).
				SetExpansion(1).
				SetReference(file))
	}
}

func (view *Files) Name() string {
	return view.name
}
