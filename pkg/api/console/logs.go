package console

import (
	"strings"

	"github.com/Br0ce/cctl/pkg/client"
	"github.com/rivo/tview"
)

const logsPage = "logs"

func CreateLogsView() *tview.TextView {
	logsView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	logsView.SetBorder(true).SetTitle(" Logs ")
	return logsView
}

func PopulateLogsView(view *tview.TextView, logs client.LogSeq) {
	var sb strings.Builder
	for line, err := range logs {
		if err != nil {
			sb.WriteString("[red]error: " + err.Error() + "[-]\n")
			break
		}
		sb.WriteString(tview.TranslateANSI(line) + "\n")
	}
	view.SetText(sb.String())
}
