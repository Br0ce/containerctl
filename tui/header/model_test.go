package header

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
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

func TestViewContainer(t *testing.T) {
	m := New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts).
		SetContainer(container.Short{Name: "/web", Image: "nginx"})

	view := ansi.Strip(m.View())
	for _, want := range []string{"Info", "Name    web", "Image   nginx"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Running") {
		t.Errorf("view shows counts with a container set:\n%s", view)
	}
	if got := lipgloss.Height(m.View()); got != Height {
		t.Errorf("view height = %d, want %d", got, Height)
	}

	view = ansi.Strip(m.ClearContainer().View())
	if !strings.Contains(view, "Total   4") || strings.Contains(view, "nginx") {
		t.Errorf("expected counts after ClearContainer:\n%s", view)
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
