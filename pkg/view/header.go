package view

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// KeyBinding describes a single key binding for display in the header.
type KeyBinding struct {
	Key  tcell.Key // Use tcell.KeyRune for printable characters.
	Rune rune      // Only meaningful when Key == tcell.KeyRune.
	Desc string
}

// InfoItem is a single key-value entry for display in the Info panel.
type InfoItem struct {
	Key   string
	Value string
}

// Page is implemented by all body views to expose their name and active key bindings.
type Page interface {
	Name() string
	KeyBindings() []KeyBinding
	InfoHeader() []InfoItem
}

type Header struct {
	name        string
	keyBindings *tview.TextView
	infoView    *tview.TextView
	*tview.Flex
}

func NewHeader(dhost, dversion string) *Header {
	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("[dodgerblue]Host:    [silver]%s[dodgerblue]\nVersion: [silver]%s", dhost, dversion)).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorDodgerBlue).
		SetDynamicColors(true)
	contextView.SetTitle(" Docker Engine API ").SetBorder(true).SetTitleColor(tcell.ColorDodgerBlue)

	keyBindings := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)
	keyBindings.SetTitle(" Key Bindings ").SetBorder(true).SetTitleColor(tcell.ColorDodgerBlue)

	infoView := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorDodgerBlue).
		SetDynamicColors(true)
	infoView.SetTitle(" Info ").SetBorder(true).SetTitleColor(tcell.ColorDodgerBlue)

	return &Header{
		name:        "header",
		keyBindings: keyBindings,
		infoView:    infoView,
		Flex: tview.NewFlex().
			AddItem(contextView, 0, 1, false).
			AddItem(keyBindings, 0, 2, false).
			AddItem(infoView, 0, 1, false),
	}
}

// SetKeyBindings updates the header display from a slice of KeyBindings.
// Views with more than 4 bindings are formatted in two columns; others single-column.
func (h *Header) SetKeyBindings(bindings []KeyBinding) {
	if len(bindings) == 0 {
		h.keyBindings.SetText("")
		return
	}

	var sb strings.Builder
	n := len(bindings)
	switch {
	case n > 6:
		// Four columns, 2 rows.
		for i := 0; i < 2; i++ {
			fmt.Fprintf(&sb, "[black:dodgerblue] %s [-:-]  [silver]%-12s", keyStr(bindings[i]), bindings[i].Desc)
			for col := 1; col < 4; col++ {
				if j := i + col*2; j < n {
					r := bindings[j]
					fmt.Fprintf(&sb, "    [black:dodgerblue] %s [-:-]  [silver]%-12s", keyStr(r), r.Desc)
				}
			}
			if i < 1 {
				sb.WriteString("\n")
			}
		}
	case n > 4:
		// Three columns, 2 rows.
		chunk := 2
		for i := 0; i < chunk; i++ {
			fmt.Fprintf(&sb, "[black:dodgerblue] %s [-:-]  [silver]%-12s", keyStr(bindings[i]), bindings[i].Desc)
			if j := i + chunk; j < n {
				r := bindings[j]
				fmt.Fprintf(&sb, "    [black:dodgerblue] %s [-:-]  [silver]%-12s", keyStr(r), r.Desc)
			}
			if k := i + 2*chunk; k < n {
				r := bindings[k]
				fmt.Fprintf(&sb, "    [black:dodgerblue] %s [-:-]  [silver]%s", keyStr(r), r.Desc)
			}
			if i < chunk-1 {
				sb.WriteString("\n")
			}
		}
	case n > 2:
		// Two columns, 2 rows.
		half := 2
		for i := 0; i < half; i++ {
			fmt.Fprintf(&sb, "[black:dodgerblue] %s [-:-]  [silver]%-12s", keyStr(bindings[i]), bindings[i].Desc)
			if j := i + half; j < n {
				r := bindings[j]
				fmt.Fprintf(&sb, "    [black:dodgerblue] %s [-:-]  [silver]%s", keyStr(r), r.Desc)
			}
			if i < half-1 {
				sb.WriteString("\n")
			}
		}
	default:
		// Single column.
		maxKeyLen := 0
		for _, b := range bindings {
			if l := len(keyStr(b)); l > maxKeyLen {
				maxKeyLen = l
			}
		}
		for i, b := range bindings {
			pad := strings.Repeat(" ", maxKeyLen-len(keyStr(b)))
			fmt.Fprintf(&sb, "[black:dodgerblue] %s%s [-:-]  [silver]%s", keyStr(b), pad, b.Desc)
			if i < n-1 {
				sb.WriteString("\n")
			}
		}
	}
	h.keyBindings.SetText(sb.String())
}

func keyStr(b KeyBinding) string {
	if b.Key == tcell.KeyRune {
		return string(b.Rune)
	}
	switch b.Key {
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyEsc:
		return "Esc"
	default:
		return fmt.Sprintf("key(%d)", b.Key)
	}
}

func (h *Header) SetInfo(items []InfoItem) {
	if len(items) == 0 {
		h.infoView.SetText("")
		return
	}
	// Display at most two rows; each row pairs two items side by side.
	var sb strings.Builder
	rows := 2
	for row := 0; row < rows; row++ {
		if row >= len(items) {
			break
		}
		fmt.Fprintf(&sb, "[dodgerblue]%-8s[silver]%-16s", items[row].Key, items[row].Value)
		if col := row + rows; col < len(items) {
			fmt.Fprintf(&sb, "    [dodgerblue]%-8s[silver]%s", items[col].Key, items[col].Value)
		}
		if row < rows-1 {
			sb.WriteString("\n")
		}
	}
	h.infoView.SetText(sb.String())
}

func (view *Header) Name() string {
	return view.name
}
