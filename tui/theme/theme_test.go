package theme

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestBox(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		content string
		wantTop string
	}{
		{"centered", 14, "a", "╭─── Info ───╮"},
		{"odd fill", 13, "a", "╭── Info ───╮"},
		{"long content", 14, strings.Repeat("x", 30) + "\nb", "╭─── Info ───╮"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Box("Info", tt.width, tt.content)
			if top := ansi.Strip(strings.Split(got, "\n")[0]); top != tt.wantTop {
				t.Errorf("top = %q, want %q", top, tt.wantTop)
			}
			if w := lipgloss.Width(got); w != tt.width {
				t.Errorf("width = %d, want %d", w, tt.width)
			}
			if h, want := lipgloss.Height(got), strings.Count(tt.content, "\n")+1+BoxFrameHeight; h != want {
				t.Errorf("height = %d, want %d", h, want)
			}
		})
	}
}
