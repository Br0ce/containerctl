package containers

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
)

var shorts = []container.Short{
	{ID: "aaaaaaaaaaaaaaaa", Project: "shop", Name: "web", Image: "nginx", Ports: []container.Port{{Public: 8080, Private: 80, Proto: "tcp"}}, Status: "Up", Age: "2h", State: "running", Health: "healthy"},
	{ID: "bbbbbbbbbbbbbbbb", Name: "db", Image: "postgres", Status: "Exited (0)", Age: "3m", State: "exited"},
	{ID: "cccccccccccccccc", Name: "cache", Image: "redis", Status: "Up (Paused)", Age: "5d", State: "paused"},
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
	return New().SetSize(120, 20).SetShorts(shorts)
}

func TestNavigation(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{"initial", nil, "web"},
		{"down", []string{"down"}, "db"},
		{"j", []string{"j", "j"}, "cache"},
		{"clamp bottom", []string{"down", "down", "down", "down"}, "cache"},
		{"clamp top", []string{"up", "k"}, "web"},
		{"bottom", []string{"G"}, "cache"},
		{"top", []string{"G", "g"}, "web"},
		// f and u are reserved for container actions and must not page.
		{"f does not page", []string{"f"}, "web"},
		{"u does not page", []string{"G", "u"}, "cache"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			for _, k := range tt.keys {
				m, _ = m.Update(press(k))
			}
			s, ok := m.Selected()
			if !ok || s.Name != tt.want {
				t.Errorf("selected = %q, want %q", s.Name, tt.want)
			}
		})
	}
}

func TestSetShortsKeepsSelection(t *testing.T) {
	m := newTestModel()
	m, _ = m.Update(press("down")) // db

	m = m.SetShorts([]container.Short{shorts[2], shorts[1], shorts[0]})
	if s, _ := m.Selected(); s.Name != "db" {
		t.Errorf("selected after reorder = %q, want db", s.Name)
	}

	m = m.SetShorts([]container.Short{shorts[0]})
	if s, _ := m.Selected(); s.Name != "web" {
		t.Errorf("selected after removal = %q, want web", s.Name)
	}

	m = m.SetShorts(nil)
	if _, ok := m.Selected(); ok {
		t.Error("expected no selection for empty list")
	}
}

// TestSelectionBarIntact checks that the colored state cell doesn't reset the
// selection background: the selected row may contain only one full reset, at
// the end of the row.
func TestSelectionBarIntact(t *testing.T) {
	m := newTestModel()
	lines := strings.Split(m.table.View(), "\n")
	selected := lines[1] // line 0 is the header

	if n := strings.Count(selected, ansi.ResetStyle); n != 1 {
		t.Errorf("selected row has %d full resets, want 1: %q", n, selected)
	}
	// The state dot and health mark reset the foreground to the terminal
	// default (SGR 39), so the selection must not set a foreground of its own.
	if n := foregrounds(selected); n != 2 {
		t.Errorf("selected row sets %d foreground colors, want 2 (state dot and health mark): %q", n, selected)
	}
	row := strings.TrimSpace(ansi.Strip(selected))
	fields := strings.Fields(row)
	if fields[0] != "●" || !slices.Equal(fields[len(fields)-4:], []string{"✓", "Up", "2h", "8080→80"}) {
		t.Errorf("selected row = %q, want it to start with the state dot and end with health, status, age and ports", row)
	}
}

// TestDimNotRunning checks that rows of containers that aren't running are
// faint on every text cell but the state dot, and that a dimmed row keeps the
// selection bar intact when selected.
func TestDimNotRunning(t *testing.T) {
	faint := ansi.Style{}.Faint().String()
	m := newTestModel()
	m, _ = m.Update(press("down")) // db, exited
	lines := strings.Split(m.table.View(), "\n")
	for i, s := range shorts {
		line := lines[i+1] // line 0 is the header
		n := strings.Count(line, faint)
		switch {
		case s.State == "running" && n != 0:
			t.Errorf("%s: running row has %d faint cells, want 0: %q", s.Name, n, line)
		case s.State != "running" && n != 7:
			t.Errorf("%s: %s row has %d faint cells, want 7: %q", s.Name, s.State, n, line)
		case n > 0 && strings.Index(line, faint) < strings.Index(line, "●"):
			t.Errorf("%s: state dot is faint: %q", s.Name, line)
		}
	}
	if n := strings.Count(lines[2], ansi.ResetStyle); n != 1 {
		t.Errorf("selected dimmed row has %d full resets, want 1: %q", n, lines[2])
	}
}

// foregrounds counts the SGR sequences in s that set a foreground color.
func foregrounds(s string) int {
	n := 0
	for _, m := range sgr.FindAllStringSubmatch(s, -1) {
		params := strings.Split(m[1], ";")
		for i := 0; i < len(params); i++ {
			switch params[i] {
			case "38":
				n++
				// Skip the color arguments: 5;n or 2;r;g;b.
				if i+1 < len(params) && params[i+1] == "5" {
					i += 2
				} else {
					i += 4
				}
			case "48", "58":
				if i+1 < len(params) && params[i+1] == "5" {
					i += 2
				} else {
					i += 4
				}
			}
		}
	}
	return n
}

var sgr = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// TestColumnsVisible checks that the state dot leads every row, STATUS and
// AGE are never cut off and the rows span the full pane width, down to the
// narrowest width where all columns but COMPOSE fit.
func TestColumnsVisible(t *testing.T) {
	// The longest status and age Docker commonly reports.
	restarting := container.Short{ID: "dddddddddddddddd", Name: "api", Image: "app", Status: "Restarting (137)", Age: "11mo", State: "restarting"}
	rows := append([]container.Short{restarting}, shorts...)
	minWidth := stateWidth + idWidth + healthWidth + statusWidth + ageWidth + 3*minFlexWidth + 8*2 + 2
	for _, w := range []int{minWidth, 80, 86, 87, 100, 120, 200} {
		m := New().SetSize(w, 10).SetShorts(rows)
		lines := strings.Split(m.table.View(), "\n")
		for i, s := range rows {
			line := ansi.Strip(lines[i+1])
			if !strings.HasPrefix(strings.TrimSpace(line), "●") {
				t.Errorf("width %d: row %q doesn't start with the state dot", w, line)
			}
			if !strings.Contains(line, s.Status) {
				t.Errorf("width %d: row %q misses status %q", w, line, s.Status)
			}
			if !strings.Contains(line, s.Age) {
				t.Errorf("width %d: row %q misses age %q", w, line, s.Age)
			}
			if got := ansi.StringWidth(line); got != w-2 {
				t.Errorf("width %d: row is %d wide, want %d", w, got, w-2)
			}
		}
	}
}

// TestComposeColumn checks that COMPOSE is shown from the width where all
// flex columns get their minimum width.
func TestComposeColumn(t *testing.T) {
	minWidth := stateWidth + idWidth + healthWidth + statusWidth + ageWidth + 4*minFlexWidth + 9*2 + 2
	for w, want := range map[int]bool{minWidth - 1: false, minWidth: true, 200: true} {
		m := New().SetSize(w, 10).SetShorts(shorts)
		lines := strings.Split(ansi.Strip(m.table.View()), "\n")
		if got := strings.Contains(lines[0], "COMPOSE"); got != want {
			t.Errorf("width %d: COMPOSE shown = %t, want %t", w, got, want)
		}
		if got := strings.Contains(lines[1], "shop"); got != want {
			t.Errorf("width %d: project shown = %t, want %t: %q", w, got, want, lines[1])
		}
	}
}

func TestHealthMark(t *testing.T) {
	tests := []struct {
		state, health, want string
	}{
		{"running", "healthy", "✓"},
		{"running", "unhealthy", "✗"},
		{"running", "starting", "…"},
		{"running", "", "?"},
		{"running", "none", "?"},
		{"exited", "healthy", ""},
	}
	for _, tt := range tests {
		r := row(container.Short{ID: "a", Status: "Up", State: tt.state, Health: tt.health})
		if got := ansi.Strip(r[5]); got != tt.want {
			t.Errorf("%s/%q: health cell = %q, want %q", tt.state, tt.health, got, tt.want)
		}
	}
}

func TestFormatPorts(t *testing.T) {
	ports := []container.Port{
		{Public: 53, Private: 53, Proto: "udp"},
		{Public: 5432, Private: 5432, Proto: "tcp"},
		{Public: 8888, Private: 8080, Proto: "tcp"},
	}
	if got, want := formatPorts(ports), "53/udp, 5432, 8888→8080"; got != want {
		t.Errorf("formatPorts = %q, want %q", got, want)
	}
	if got := formatPorts(nil); got != "" {
		t.Errorf("formatPorts(nil) = %q, want empty", got)
	}
}

func TestViewFitsSize(t *testing.T) {
	long := container.Short{
		ID:    "dddddddddddddddd",
		Name:  "a-very-long-container-name-that-overflows",
		Image: "registry.example.com/some/very/long/image:tag",
		State: "running",
	}
	rows := append([]container.Short{long}, shorts...)
	for _, size := range [][2]int{{120, 10}, {200, 30}} {
		m := New().SetSize(size[0], size[1]).SetShorts(rows)
		v := m.View()
		if w, h := lipgloss.Width(v), lipgloss.Height(v); w != size[0] || h != size[1] {
			t.Errorf("size %v: view is %dx%d", size, w, h)
		}
	}
}
