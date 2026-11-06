package preview

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/file"
)

var script = file.Info{Name: "app.py", Path: "/app/app.py"}

func newTestModel() Model {
	return New().SetSize(60, 10).Start(script)
}

func TestLoading(t *testing.T) {
	got := ansi.Strip(newTestModel().View())
	if !strings.Contains(got, "app.py") || !strings.Contains(got, "Loading…") {
		t.Errorf("expected title and loading note: %q", got)
	}
}

func TestText(t *testing.T) {
	got := ansi.Strip(newTestModel().SetContent([]byte("import os\r\nprint(os.getcwd())\n"), false).View())
	for _, want := range []string{"import os", "print(os.getcwd())"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if strings.Contains(got, "truncated") {
		t.Errorf("untruncated file marked as truncated: %q", got)
	}
}

func TestTextLinesSanitized(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"escape sequences stripped", "\x1b[31mred\x1b[0m \x1b]0;title\x07done", []string{"red done"}},
		{"control characters replaced", "a\rb\x00c\x7fd", []string{"a·b·c·d"}},
		{"tabs expanded", "\tx\nab\tc\nü\tc", []string{"        x", "ab      c", "ü       c"}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textLines(tt.in)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("textLines(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNoEscapeReachesView(t *testing.T) {
	m := newTestModel().SetContent([]byte("\x1b]52;c;cHduZWQ=\x07\x1b[2J"), false)
	if v := m.viewport.View(); strings.Contains(v, "]52") || strings.Contains(v, "[2J") {
		t.Errorf("escape sequence reached the view: %q", v)
	}
}

func TestBinary(t *testing.T) {
	got := ansi.Strip(newTestModel().SetSize(100, 10).SetContent([]byte("\x7fELF\x02\x01\x00\x00"), false).View())
	for _, want := range []string{"Binary file (application/octet-stream)", "7f 45 4c 46", "|.ELF....|"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
}

func TestImage(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	got := ansi.Strip(newTestModel().SetContent(png, false).View())
	if !strings.Contains(got, "Image (image/png), no preview yet") {
		t.Errorf("expected image note: %q", got)
	}
}

func TestEmpty(t *testing.T) {
	if got := ansi.Strip(newTestModel().SetContent(nil, false).View()); !strings.Contains(got, "Empty file") {
		t.Errorf("expected empty note: %q", got)
	}
}

func TestTruncated(t *testing.T) {
	// The read cut the trailing "ü" (c3 bc) in half; it is still text.
	data := []byte("line\n\xc3")
	got := ansi.Strip(newTestModel().SetContent(data, true).View())
	if !strings.Contains(got, "line") || !strings.Contains(got, "… truncated after 6 B") || strings.Contains(got, "Binary") {
		t.Errorf("expected text with truncation note: %q", got)
	}
}

func TestErr(t *testing.T) {
	m := newTestModel().SetContent([]byte("old"), false).SetErr(errors.New("is a directory"))
	got := ansi.Strip(m.View())
	if !strings.Contains(got, "Error: is a directory") || strings.Contains(got, "old") {
		t.Errorf("expected only the error: %q", got)
	}
}

func TestStartResetsScroll(t *testing.T) {
	m := newTestModel().SetContent([]byte(strings.Repeat("x\n", 50)), false)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.viewport.YOffset() == 0 {
		t.Fatal("G did not scroll")
	}
	if m = m.Start(file.Info{Name: "other"}); m.viewport.YOffset() != 0 {
		t.Error("scroll position kept for a new file")
	}
}

func TestFocusBorder(t *testing.T) {
	m := newTestModel()
	if m.SetFocused(true).View() == m.SetFocused(false).View() {
		t.Error("focused and unfocused pane look the same")
	}
}

func TestViewFitsSize(t *testing.T) {
	long := []byte(strings.Repeat("y", 300) + "\n" + strings.Repeat("z\n", 40))
	for _, size := range [][2]int{{30, 5}, {60, 10}, {120, 30}} {
		m := New().SetSize(size[0], size[1]).Start(script)
		for _, mm := range []Model{m, m.SetContent(long, true), m.SetErr(errors.New("boom"))} {
			v := mm.View()
			if w, h := lipgloss.Width(v), lipgloss.Height(v); w != size[0] || h != size[1] {
				t.Errorf("size %v: view is %dx%d", size, w, h)
			}
		}
	}
}
