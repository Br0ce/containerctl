package view

import (
	"strings"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/rivo/tview"
)

const Log = "logs"

func NewLog() *tview.TextView {
	logsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	logsView.SetBorder(true).SetTitle(" Logs ")
	return logsView
}

func PopulateLog(view *tview.TextView, logs client.LogSeq) {
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
