package inspect

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/tui/theme"
)

var now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

var details = container.Details{
	ID:            "aaaaaaaaaaaaaaaa",
	Name:          "web",
	Image:         "nginx:1.27",
	State:         "exited",
	ExitCode:      137,
	OOMKilled:     true,
	StartedAt:     now.Add(-3 * time.Hour),
	FinishedAt:    now.Add(-30 * time.Second),
	RestartPolicy: "on-failure:5",
	RestartCount:  2,
	Entrypoint:    []string{"/docker-entrypoint.sh"},
	Cmd:           []string{"nginx", "-g", "daemon off;"},
	Health: &container.Health{
		Status:        "unhealthy",
		FailingStreak: 3,
		Checks: []container.HealthCheck{
			{Start: now.Add(-20 * time.Second), ExitCode: 0, Output: "ok"},
			{Start: now.Add(-10 * time.Second), ExitCode: 1, Output: "curl: (7) refused\nretrying"},
		},
	},
	Networks: []container.Network{{Name: "myproject_default_network", IP: "172.18.0.2/16", Aliases: []string{"web"}}},
	Ports:    []container.Port{{Public: 8080, Private: 80, Proto: "tcp"}, {Private: 443, Proto: "tcp"}},
	Mounts:   []container.Mount{{Type: "volume", Source: "data", Destination: "/data", RW: true}},
	Resources: container.Resources{
		Memory:   1536 << 20,
		NanoCPUs: 1_500_000_000,
	},
	Env:    []string{"PATH=/usr/bin", "EMPTY=", "SECRET=hunter2"},
	Labels: map[string]string{"z": "last", "a": "first"},
}

func TestRender(t *testing.T) {
	got := ansi.Strip(strings.Join(render(details, false, theme.Header, now), "\n"))
	for _, want := range []string{
		"Restart policy  on-failure:5",
		"Exit code       137",
		"OOM killed      yes",
		"(30s ago)",
		`Command         nginx -g "daemon off;"`,
		"Failing streak  3",
		"exit 1  curl: (7) refused",
		"retrying",
		"myproject_default_network IP 172.18.0.2/16",
		"Aliases web",
		"Ports           8080 → 80/tcp",
		"443/tcp (not published)",
		"volume          data → /data (rw)",
		"Memory          1.5 GiB",
		"CPUs            1.5",
		"PATH=••••••",
		"EMPTY=\n",
		"a=first\n  z=last",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "hunter2") {
		t.Error("environment value shown while masked")
	}
	// The latest check comes first.
	if strings.Index(got, "exit 1") > strings.Index(got, "exit 0") {
		t.Error("expected the latest check first")
	}

	got = ansi.Strip(strings.Join(render(details, true, theme.Header, now), "\n"))
	if !strings.Contains(got, "SECRET=hunter2") {
		t.Errorf("environment not shown when toggled:\n%s", got)
	}
}

func TestRenderLeavesOutUnsetSections(t *testing.T) {
	got := ansi.Strip(strings.Join(render(container.Details{Name: "web", State: "running"}, false, theme.Header, now), "\n"))
	for _, missing := range []string{"HEALTH", "NETWORK", "MOUNTS", "ENVIRONMENT", "LABELS", "Finished", "Exit code"} {
		if strings.Contains(got, missing) {
			t.Errorf("render contains %q:\n%s", missing, got)
		}
	}
	if !strings.Contains(got, "No limits set") {
		t.Errorf("expected a note on missing limits:\n%s", got)
	}
}

func TestPane(t *testing.T) {
	m := New().SetSize(80, 10).Start()
	if got := ansi.Strip(m.View()); !strings.Contains(got, "Inspect") || !strings.Contains(got, "Loading…") {
		t.Errorf("expected the loading state: %q", got)
	}

	m = m.SetDetails(details)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if got := ansi.Strip(m.View()); !strings.Contains(got, "SECRET=••••••") {
		t.Errorf("expected the masked env at the end: %q", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if got := ansi.Strip(m.View()); !strings.Contains(got, "SECRET=hunter2") {
		t.Errorf("expected the env after 'e': %q", got)
	}
	if h := m.KeyMap().Env.Help().Desc; h != "hide env" {
		t.Errorf("env help = %q, want hide env", h)
	}

	// A refresh keeps the scroll position, a failed one keeps the details.
	m = m.SetDetails(details).SetErr(errors.New("boom"))
	got := ansi.Strip(m.View())
	if !strings.Contains(got, "SECRET=hunter2") {
		t.Errorf("expected the scroll position kept: %q", got)
	}
	if h := lipgloss.Height(got); h != 10 {
		t.Errorf("height = %d, want 10", h)
	}

	m = m.Start()
	m = m.SetDetails(details)
	if got := ansi.Strip(m.View()); !strings.Contains(got, "GENERAL") {
		t.Errorf("expected Start to scroll back to the top: %q", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if got := ansi.Strip(m.View()); !strings.Contains(got, "SECRET=••••••") {
		t.Errorf("expected Start to mask the env again: %q", got)
	}
}

func TestSectionTitlesFollowBackground(t *testing.T) {
	for _, dark := range []bool{true, false} {
		header := theme.HeaderFor(dark)
		if got, want := render(details, false, header, now)[0], header.Render("GENERAL"); got != want {
			t.Errorf("dark=%v: title = %q, want %q like the column titles", dark, got, want)
		}
	}
	m := New().SetSize(80, 10).Start().SetDetails(details)
	if m.SetDark(true).View() == m.SetDark(false).View() {
		t.Error("SetDark doesn't restyle the shown details")
	}
}
