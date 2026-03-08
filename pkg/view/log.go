package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Br0ce/containerctl/pkg/client"
)

type Log struct {
	name string
	*tview.TextView
}

func NewLog() *Log {
	logsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	logsView.SetBorder(true).SetTitle(" Logs ").SetTitleColor(tcell.ColorAqua)
	return &Log{
		name:     "log",
		TextView: logsView,
	}
}

func (view *Log) Populate(logs client.LogSeq) {
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
}

func (view *Log) Name() string {
	return view.name
}
