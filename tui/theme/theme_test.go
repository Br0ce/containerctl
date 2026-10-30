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

func TestStateCell(t *testing.T) {
	seen := map[string]string{}
	for _, state := range []string{"running", "paused", "dead", "created", "exited"} {
		got := StateCell(state)
		if s := ansi.Strip(got); s != "●" {
			t.Errorf("%s: cell text = %q, want ●", state, s)
		}
		if other, ok := seen[got]; ok {
			t.Errorf("%s and %s share a color", state, other)
		}
		seen[got] = state
	}
}

func TestHealthMark(t *testing.T) {
	tests := map[string]string{"healthy": "✓", "unhealthy": "✗", "starting": "…", "": "?", "none": "?"}
	for health, want := range tests {
		if got := ansi.Strip(HealthMark(health)); got != want {
			t.Errorf("%q: mark = %q, want %q", health, got, want)
		}
	}
}

func TestHeaderFor(t *testing.T) {
	if got := HeaderFor(false).GetForeground(); got != titleLight {
		t.Errorf("light header = %v, want dark teal %v", got, titleLight)
	}
	if got := HeaderFor(true).GetForeground(); got != titleDark {
		t.Errorf("dark header = %v, want teal %v", got, titleDark)
	}
}

func TestBytes(t *testing.T) {
	for b, want := range map[uint64]string{
		512:        "512 B",
		1024:       "1 KiB",
		256 << 20:  "256 MiB",
		1536 << 20: "1.5 GiB",
		4 << 40:    "4 TiB",
	} {
		if got := Bytes(b); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", b, got, want)
		}
	}
}
