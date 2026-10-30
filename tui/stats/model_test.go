package stats

import (
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
)

var read = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

var sample = container.Stats{
	Read:       read,
	CPUPercent: 50,
	OnlineCPUs: 4,
	MemUsage:   256 << 20,
	MemLimit:   1 << 30,
	NetRx:      1 << 20,
	NetTx:      2048,
}

func newTestModel() Model {
	return New().SetSize(80, 14).Start()
}

func TestLoading(t *testing.T) {
	got := ansi.Strip(newTestModel().View())
	if !strings.Contains(got, "Stats") || !strings.Contains(got, "Loading…") {
		t.Errorf("expected title and loading: %q", got)
	}
	if h := lipgloss.Height(got); h != 14 {
		t.Errorf("height = %d, want 14", h)
	}
}

func TestSetStats(t *testing.T) {
	m := newTestModel().SetStats(sample)
	got := ansi.Strip(m.View())
	for _, want := range []string{"CPU", " 50.0%", "of 4 CPUs", "MEMORY", " 25.0%", "256 MiB / 1 GiB", "NETWORK", "Received        1 MiB", "Sent            2 KiB"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if strings.Contains(got, "/s") {
		t.Errorf("rates shown without a previous sample: %q", got)
	}

	next := sample
	next.Read = read.Add(2 * time.Second)
	next.NetRx += 4096
	m = m.SetStats(next)
	got = ansi.Strip(m.View())
	for _, want := range []string{"2 KiB/s", "0 B/s"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain rate %q: %q", want, got)
		}
	}
}

func TestNotRunning(t *testing.T) {
	m := newTestModel().SetStats(sample).SetStats(container.Stats{})
	if got := ansi.Strip(m.View()); !strings.Contains(got, "not running") || strings.Contains(got, "CPU") {
		t.Errorf("expected the not running notice: %q", got)
	}
	// The first sample after a restart has no rates, since the previous one
	// is empty.
	next := sample
	next.Read = read.Add(time.Minute)
	if got := ansi.Strip(m.SetStats(next).View()); strings.Contains(got, "/s") {
		t.Errorf("rates derived from an empty sample: %q", got)
	}
}

func TestErrKeepsStats(t *testing.T) {
	m := newTestModel().SetStats(sample).SetErr(errors.New("boom"))
	got := ansi.Strip(m.View())
	if !strings.Contains(got, "Error: boom") || !strings.Contains(got, "256 MiB") {
		t.Errorf("expected the error above the last sample: %q", got)
	}
	if got := ansi.Strip(m.Start().View()); strings.Contains(got, "boom") || strings.Contains(got, "256 MiB") {
		t.Errorf("Start must clear the pane: %q", got)
	}
}

func TestBar(t *testing.T) {
	for share, want := range map[float64]string{
		0:    "░░░░",
		0.5:  "██░░",
		1:    "████",
		1.7:  "████",
		-0.1: "░░░░",
	} {
		if got := ansi.Strip(bar(share, 4)); got != want {
			t.Errorf("bar(%v) = %q, want %q", share, got, want)
		}
	}
}

func TestRate(t *testing.T) {
	if got := rate(0, 3072, 2*time.Second); got != "1.5 KiB/s" {
		t.Errorf("rate() = %q, want 1.5 KiB/s", got)
	}
	if got := rate(100, 10, time.Second); got != "0 B/s" {
		t.Errorf("rate() after reset = %q, want 0 B/s", got)
	}
}

func TestSectionTitlesFollowBackground(t *testing.T) {
	m := newTestModel().SetStats(sample)
	if m.SetDark(true).View() == m.SetDark(false).View() {
		t.Error("SetDark doesn't restyle the shown stats")
	}
}
