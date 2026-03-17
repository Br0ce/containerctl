package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/client"
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

func (view *Log) Populate(logs client.LogSeq, short container.Short) {
	var sb strings.Builder
	for line, err := range logs {
		if err != nil {
			sb.WriteString("[red]error: " + err.Error() + "[-]\n")
			break
		}
		sb.WriteString(tview.TranslateANSI(line) + "\n")
	}
	view.SetText(sb.String())
	view.ScrollToEnd()
	view.curShort = &short
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
