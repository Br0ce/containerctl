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

// Page is implemented by all body views to expose their name and active key bindings.
type Page interface {
	Name() string
	KeyBindings() []KeyBinding
}

type Header struct {
	name        string
	keyBindings *tview.TextView
	*tview.Flex
}

func NewHeader(dhost, dversion string) *Header {
	contextView := tview.NewTextView().
		SetText(fmt.Sprintf("Docker Engine API Host:    [white]%s[aqua]\nDocker Engine API Version: [white]%s", dhost, dversion)).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorAqua).
		SetDynamicColors(true)

	appTitle := tview.NewTextView().
		SetText(`                 _        _                     _   _ 
  ___ ___  _ __ | |_ __ _(_)_ __   ___ _ __ ___| |_| |
 / __/ _ \| '_ \| __/ _` + "`" + ` | | '_ \ / _ \ '__/ __| __| |
| (_| (_) | | | | || (_| | | | | |  __/ | | (__| |_| |
 \___\___/|_| |_|\__\__,_|_|_| |_|\___|_|  \___|\__|_|`).
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorMediumBlue)

	keyBindings := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	return &Header{
		name:        "header",
		keyBindings: keyBindings,
		Flex: tview.NewFlex().
			AddItem(contextView, 0, 1, false).
			AddItem(keyBindings, 0, 1, false).
			AddItem(appTitle, 0, 1, false),
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
	if len(bindings) > 4 {
		half := (len(bindings) + 1) / 2
		for i := 0; i < half; i++ {
			fmt.Fprintf(&sb, "[dodgerblue]<%s>[gray]   %-10s", keyStr(bindings[i]), bindings[i].Desc)
			if j := i + half; j < len(bindings) {
				r := bindings[j]
				fmt.Fprintf(&sb, "  [dodgerblue]<%s>[gray]   %s", keyStr(r), r.Desc)
			}
			if i < half-1 {
				sb.WriteString("\n")
			}
		}
	} else {
		maxKeyLen := 0
		for _, b := range bindings {
			if l := len(keyStr(b)); l > maxKeyLen {
				maxKeyLen = l
			}
		}
		for i, b := range bindings {
			pad := strings.Repeat(" ", maxKeyLen-len(keyStr(b)))
			fmt.Fprintf(&sb, "[dodgerblue]<%s>%s[gray]   %s", keyStr(b), pad, b.Desc)
			if i < len(bindings)-1 {
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

func (view *Header) Name() string {
	return view.name
}
