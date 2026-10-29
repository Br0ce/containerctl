package logs

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func numbered(from, to int) []string {
	var lines []string
	for i := from; i < to; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	return lines
}

func newTestModel() Model {
	return New().SetSize(80, 7).Start() // 5 visible lines
}

func TestTitle(t *testing.T) {
	got := newTestModel().View()
	if !strings.Contains(got, "Logs") {
		t.Errorf("title missing: %q", got)
	}
	if h := lipgloss.Height(got); h != 7 {
		t.Errorf("height = %d, want 7", h)
	}
}

func TestAppendFollows(t *testing.T) {
	m := newTestModel().Append(numbered(0, 10)...)
	m = m.Append("line 10")
	got := m.View()
	if !strings.Contains(got, "line 10") || strings.Contains(got, "line 5") {
		t.Errorf("expected the last 5 lines: %q", got)
	}
}

func TestAppendKeepsScrollPosition(t *testing.T) {
	m := newTestModel().Append(numbered(0, 10)...)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m = m.Append("line 10")
	got := m.View()
	if !strings.Contains(got, "line 0") || strings.Contains(got, "line 10") {
		t.Errorf("expected the first lines: %q", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	m = m.Append("line 11")
	if !strings.Contains(m.View(), "line 11") {
		t.Errorf("expected to follow again after G: %q", m.View())
	}
}

func TestAppendDropsOldLines(t *testing.T) {
	m := newTestModel().Append(numbered(0, maxLines+10)...)
	if len(m.lines) != maxLines {
		t.Fatalf("kept %d lines, want %d", len(m.lines), maxLines)
	}
	if m.lines[0] != "line 10" {
		t.Errorf("first line = %q, want line 10", m.lines[0])
	}
}

func TestStartClears(t *testing.T) {
	m := newTestModel().Append("old").Start()
	if got := m.View(); strings.Contains(got, "old") {
		t.Errorf("expected an empty session: %q", got)
	}
}

func TestCarriageReturnRemoved(t *testing.T) {
	m := newTestModel().Append("progress\r50%")
	if strings.Contains(m.View(), "\r") {
		t.Error("view contains a carriage return")
	}
}
