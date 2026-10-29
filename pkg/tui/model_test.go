package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/container"
)

type fakeClient struct {
	shorts  []container.Short
	err     error
	logs    []string
	logsErr error
	logsCtx context.Context
}

func (f *fakeClient) AllShorts(ctx context.Context) ([]container.Short, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("expected a deadline on the request context")
	}
	return f.shorts, f.err
}

var shorts = []container.Short{
	{ID: "aaaaaaaaaaaaaaaa", Name: "web", Image: "nginx", Status: "Up", State: "running"},
}

func (f *fakeClient) Logs(ctx context.Context, _ string) (client.LogSeq, context.CancelFunc) {
	f.logsCtx = ctx
	return func(yield func(string, error) bool) {
		for _, l := range f.logs {
			if !yield(l, nil) {
				return
			}
		}
		if f.logsErr != nil {
			yield("", f.logsErr)
		}
	}, func() {}
}

func (f *fakeClient) DaemonHost() string    { return "unix:///var/run/docker.sock" }
func (f *fakeClient) DaemonVersion() string { return "1.47" }

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func TestInitLoadsContainers(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})

	m, _ = update(m, m.Init()())
	if m.err != nil {
		t.Fatalf("unexpected error: %v", m.err)
	}
	if s, ok := m.containers.Selected(); !ok || s.ID != shorts[0].ID {
		t.Errorf("selected = %+v, want %s", s, shorts[0].ID)
	}
}

func TestFetchError(t *testing.T) {
	m := newModel(&fakeClient{err: errors.New("boom")})
	m, _ = update(m, m.Init()())
	if !strings.Contains(m.View().Content, "boom") {
		t.Errorf("view does not show the error: %q", m.View().Content)
	}
}

func TestQuit(t *testing.T) {
	m := newModel(&fakeClient{})
	_, cmd := update(m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}

func TestContainersMsgSchedulesTick(t *testing.T) {
	m := newModel(&fakeClient{})
	for _, msg := range []containersMsg{{shorts: shorts}, {err: errors.New("boom")}} {
		if _, cmd := update(m, msg); cmd == nil {
			t.Errorf("expected tick command after %+v", msg)
		}
	}
}

func TestTickFetchesContainers(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	_, cmd := update(m, tickMsg{})
	if cmd == nil {
		t.Fatal("expected fetch command")
	}
	msg, ok := cmd().(containersMsg)
	if !ok {
		t.Fatalf("expected containersMsg, got %T", msg)
	}
	if len(msg.shorts) != len(shorts) {
		t.Errorf("got %d shorts, want %d", len(msg.shorts), len(shorts))
	}
}

func TestHelpToggle(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m, _ = update(m, m.Init()())

	got := m.View().Content
	if !strings.Contains(got, "go to start") {
		t.Fatalf("full help expected by default: %q", got)
	}
	if h := lipgloss.Height(m.footer()); h != 2 {
		t.Errorf("full help height = %d, want 2", h)
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if got := m.View().Content; !strings.Contains(got, "toggle help") || strings.Contains(got, "go to start") {
		t.Errorf("short help expected after '?': %q", got)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if !strings.Contains(m.View().Content, "go to start") {
		t.Error("full help not shown after second '?'")
	}
}

// openLogs loads the containers and presses l on the selected one.
func openLogs(t *testing.T, cli *fakeClient) (Model, tea.Cmd) {
	t.Helper()
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m, _ = update(m, m.Init()())
	m, cmd := update(m, tea.KeyPressMsg{Code: 'l', Text: "l"})
	if m.page != logsPage {
		t.Fatal("expected logs page after 'l'")
	}
	if cmd == nil {
		t.Fatal("expected logs command")
	}
	return m, cmd
}

func TestOpenLogsStreamsLines(t *testing.T) {
	m, cmd := openLogs(t, &fakeClient{shorts: shorts, logs: []string{"first line", "second line"}})

	msg, ok := cmd().(logsMsg)
	if !ok {
		t.Fatalf("expected logsMsg, got %T", msg)
	}
	m, cmd = update(m, msg)
	got := ansi.Strip(m.View().Content)
	for _, want := range []string{"Name    web", "Image   nginx", "first line", "second line", "esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	// Drain the stream until it reports its end.
	for cmd != nil {
		msg := cmd().(logsMsg)
		m, cmd = update(m, msg)
		if !msg.done && cmd == nil {
			t.Fatal("expected the stream to be awaited again")
		}
	}
}

func TestLogsError(t *testing.T) {
	m, cmd := openLogs(t, &fakeClient{shorts: shorts, logsErr: errors.New("boom")})
	for cmd != nil {
		m, cmd = update(m, cmd())
	}
	if !strings.Contains(m.View().Content, "Error: boom") {
		t.Errorf("view does not show the error: %q", m.View().Content)
	}
}

func TestLogsBack(t *testing.T) {
	cli := &fakeClient{shorts: shorts}
	m, _ := openLogs(t, cli)

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.page != containersPage {
		t.Error("expected containers page after esc")
	}
	if cli.logsCtx.Err() == nil {
		t.Error("expected log stream to be canceled")
	}
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Containers") || !strings.Contains(got, "Total") || strings.Contains(got, "Image   nginx") {
		t.Errorf("containers pane and counts not shown: %q", got)
	}
}

func TestStaleLogsDropped(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})

	m, cmd := update(m, logsMsg{session: m.logSession - 1, lines: []string{"stale"}})
	if cmd != nil {
		t.Error("stale stream must not be awaited again")
	}
	if strings.Contains(m.View().Content, "stale") {
		t.Error("view shows lines of a stale stream")
	}
}

func TestLogsKeyIgnoredOnLogsPage(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})
	session := m.logSession
	m, _ = update(m, tea.KeyPressMsg{Code: 'l', Text: "l"})
	if m.logSession != session {
		t.Error("'l' on the logs page must not start a new stream")
	}
}
