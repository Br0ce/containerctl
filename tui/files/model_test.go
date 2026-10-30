package files

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/file"
)

var listing = []file.Info{
	{Name: "..", Path: "/", IsDir: true},
	{Name: "etc", Path: "/app/etc", IsDir: true},
	{Name: "app.log", Path: "/app/app.log", Size: 3 * 1024 * 1024},
	{Name: "README", Path: "/app/README", Size: 42},
}

func press(k string) tea.KeyPressMsg {
	switch k {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
}

func newTestModel() Model {
	return New().SetSize(80, 10).SetFiles("/app", listing)
}

func TestView(t *testing.T) {
	got := ansi.Strip(newTestModel().View())
	for _, want := range []string{"/app", "NAME", "SIZE", "..", "etc", "app.log", "3.0 MB", "README", "42 B"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	// The table re-renders the cell end, so only the bold start is checked.
	bold := ansi.Style{}.Bold().String()
	faint := ansi.Style{}.Faint().String()
	view := newTestModel().View()
	if !strings.Contains(view, bold+"etc") || strings.Contains(view, faint+"etc") {
		t.Errorf("directory not bold: %q", view)
	}
	if !strings.Contains(view, faint+"README") || strings.Contains(view, bold+"README") {
		t.Errorf("file not faint: %q", view)
	}
}

func TestTitleBeforeListing(t *testing.T) {
	if got := ansi.Strip(New().SetSize(80, 10).Start().View()); !strings.Contains(got, "Files") {
		t.Errorf("title missing: %q", got)
	}
}

func TestNavigation(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{"initial", nil, ".."},
		{"down", []string{"down"}, "etc"},
		{"bottom", []string{"G"}, "README"},
		{"top", []string{"G", "g"}, ".."},
		// f and u are reserved for container actions and must not page.
		{"f does not page", []string{"f"}, ".."},
		{"u does not page", []string{"G", "u"}, "README"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			for _, k := range tt.keys {
				m, _ = m.Update(press(k))
			}
			f, ok := m.Selected()
			if !ok || f.Name != tt.want {
				t.Errorf("selected = %q, want %q", f.Name, tt.want)
			}
		})
	}
}

func TestSetFilesResetsCursorAndError(t *testing.T) {
	m := newTestModel()
	m, _ = m.Update(press("G"))
	m = m.SetErr(errors.New("boom"))

	m = m.SetFiles("/app/etc", listing[:2])
	if f, _ := m.Selected(); f.Name != ".." {
		t.Errorf("selected = %q, want ..", f.Name)
	}
	if strings.Contains(m.View(), "boom") {
		t.Error("error not cleared by a new listing")
	}

	m = m.SetFiles("/", nil)
	if _, ok := m.Selected(); ok {
		t.Error("expected no selection for empty listing")
	}
}

// TestSelectionBarIntact checks that the bold directory and faint file names
// don't reset the selection colors: the selected row may contain only one
// full reset.
func TestSelectionBarIntact(t *testing.T) {
	for _, row := range []int{1, 2} { // etc, app.log
		m := newTestModel()
		for range row {
			m, _ = m.Update(press("down"))
		}
		selected := strings.Split(m.table.View(), "\n")[row+1] // line 0 is the header
		if n := strings.Count(selected, ansi.ResetStyle); n != 1 {
			t.Errorf("row %d: selected row has %d full resets, want 1: %q", row, n, selected)
		}
	}
}

func TestViewFitsSize(t *testing.T) {
	long := file.Info{Name: strings.Repeat("x", 200), Size: 1}
	rows := append([]file.Info{long}, listing...)
	for _, size := range [][2]int{{40, 5}, {120, 10}, {200, 30}} {
		m := New().SetSize(size[0], size[1]).SetFiles("/app", rows)
		for _, err := range []error{nil, errors.New("boom")} {
			v := m.SetErr(err).View()
			if w, h := lipgloss.Width(v), lipgloss.Height(v); w != size[0] || h != size[1] {
				t.Errorf("size %v, err %v: view is %dx%d", size, err, w, h)
			}
		}
	}
}
