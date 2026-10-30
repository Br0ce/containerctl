package header

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/engine"
	"github.com/Br0ce/containerctl/tui/theme"
)

var shorts = []container.Short{
	{ID: "a", Project: "shop", State: "running", Health: "healthy"},
	{ID: "b", Project: "shop", State: "running", Health: "unhealthy"},
	{ID: "c", Project: "blog", State: "paused"},
	{ID: "d", State: "exited"},
	{ID: "e", State: "created"},
}

func TestView(t *testing.T) {
	view := ansi.Strip(New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts).View())

	for _, want := range []string{
		"Docker Engine API", "unix:///var/run/docker.sock", "1.47",
		"Info", "Running: 2", "Paused:    1", "Exited:   1",
		"Total:   5", "Unhealthy: 1", "Projects: 2",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q:\n%s", want, view)
		}
	}
}

func TestViewContainer(t *testing.T) {
	m := New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts).
		SetContainer(container.Short{Name: "web", Image: "nginx"})

	view := ansi.Strip(m.View())
	for _, want := range []string{"Info", "Name:    web", "Image:   nginx"} {
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
	if !strings.Contains(view, "Total:   5") || strings.Contains(view, "nginx") {
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

func TestUnhealthyAlert(t *testing.T) {
	// The box rewrites the resets, so only the color in front of the count is
	// compared.
	red, _, _ := strings.Cut(theme.Alert.Render("x"), "x")

	view := New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts).View()
	if !strings.Contains(view, red+"1") {
		t.Errorf("unhealthy count isn't highlighted:\n%q", view)
	}

	view = New("unix:///var/run/docker.sock", "1.47").SetWidth(120).SetShorts(shorts[2:]).View()
	if strings.Contains(view, red) {
		t.Errorf("zero unhealthy count is highlighted:\n%q", view)
	}
}

func TestViewEngine(t *testing.T) {
	m := New("ssh://deploy@prod-01", "1.47").SetWidth(140).SetShorts(shorts).SetEngine(engine.Info{
		Hostname: "prod-01", Version: "29.0.1", APIVersion: "1.52",
		OS: "linux", Arch: "arm64", CPUs: 8, Memory: 16 << 30,
	})
	view := ansi.Strip(m.View())
	for _, want := range []string{
		"Host:    prod-01 (ssh://deploy@prod-01)",
		"Engine:  29.0.1 (API 1.52) · linux/arm64 · 8 CPUs · 16 GiB",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q:\n%s", want, view)
		}
	}
	if got := lipgloss.Height(m.View()); got != Height {
		t.Errorf("view height = %d, want %d", got, Height)
	}
}

func TestEngineLine(t *testing.T) {
	tests := []struct {
		info engine.Info
		want string
	}{
		{engine.Info{Version: "29.0.1", APIVersion: "1.52", OS: "linux", Arch: "amd64", CPUs: 1, Memory: 512 << 20}, "29.0.1 (API 1.52) · linux/amd64 · 1 CPU · 512 MiB"},
		{engine.Info{Version: "29.0.1", CPUs: 4, Memory: 15_600 << 20}, "29.0.1 · 4 CPUs · 15 GiB"},
		{engine.Info{}, ""},
	}
	for _, tt := range tests {
		if got := engineLine(tt.info); got != tt.want {
			t.Errorf("engineLine(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}
