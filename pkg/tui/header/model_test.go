package header

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/pkg/container"
)

var shorts = []container.Short{
	{ID: "a", State: "running"},
	{ID: "b", State: "running"},
	{ID: "c", State: "paused"},
	{ID: "d", State: "exited"},
}

func TestView(t *testing.T) {
	view := ansi.Strip(New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts).View())

	for _, want := range []string{
		"Docker Engine API", "unix:///var/run/docker.sock", "1.47",
		"Info", "Running 2", "Paused 1", "Total   4",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q:\n%s", want, view)
		}
	}
}

func TestViewSize(t *testing.T) {
	for _, w := range []int{120, 61, 20} {
		view := New("unix:///var/run/docker.sock", "1.47").SetWidth(w).SetShorts(shorts).View()
		if got := lipgloss.Width(view); got != w {
			t.Errorf("width %d: view width = %d", w, got)
		}
		if got := lipgloss.Height(view); got != Height {
			t.Errorf("width %d: view height = %d, want %d", w, got, Height)
		}
	}
}
