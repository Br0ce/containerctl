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
	table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDodgerBlue).Foreground(tcell.ColorWhite))

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

	for col, h := range []string{"Name", "Size"} {
		view.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetExpansion(1))
	}

	view.SetTitle(fmt.Sprintf(" %s ", files[0].DisplayName))
	for row, file := range files {
		nameColor := tcell.ColorDimGray
		if file.IsDir {
			nameColor = tcell.ColorDodgerBlue
		}
		view.SetCell(row+1, 0, tview.NewTableCell(file.Name).SetTextColor(nameColor).SetReference(file))
		view.SetCell(row+1, 1, tview.NewTableCell(file.SizeString()).SetTextColor(tcell.ColorDimGray).SetReference(file))
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
