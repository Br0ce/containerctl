package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/file"
)

type fakeClient struct {
	shorts  []container.Short
	err     error
	logs    []string
	logsErr error
	logsCtx context.Context
	// files maps a directory path to its listing; "" is the working
	// directory.
	files    map[string][]file.Info
	filesErr error
	// terminal, if set, runs in place of the shell session.
	terminal func(id string, in io.Reader, out io.Writer) error
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

func (f *fakeClient) FilesIn(ctx context.Context, root file.Info) ([]file.Info, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("expected a deadline on the request context")
	}
	if root.ContainerID != shorts[0].ID {
		return nil, errors.New("unexpected container " + root.ContainerID)
	}
	return f.files[root.Path], f.filesErr
}

func (f *fakeClient) Terminal(_ context.Context, id string, in io.Reader, out io.Writer) error {
	return f.terminal(id, in, out)
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

var workdir = []file.Info{
	{Name: "..", Path: "/", IsDir: true, ContainerID: shorts[0].ID, DisplayName: "/app"},
	{Name: "static", Path: "/app/static", IsDir: true, ContainerID: shorts[0].ID, DisplayName: "/app"},
	{Name: "main.go", Path: "/app/main.go", ContainerID: shorts[0].ID, DisplayName: "/app", Size: 2048},
}

var static = []file.Info{
	{Name: "..", Path: "/app", IsDir: true, ContainerID: shorts[0].ID, DisplayName: "/app/static"},
	{Name: "index.html", Path: "/app/static/index.html", ContainerID: shorts[0].ID, DisplayName: "/app/static", Size: 12},
}

func newFilesClient() *fakeClient {
	return &fakeClient{shorts: shorts, files: map[string][]file.Info{"": workdir, "/app/static": static}}
}

// openFiles loads the containers, presses f on the selected one and delivers
// the listing.
func openFiles(t *testing.T, cli *fakeClient) Model {
	t.Helper()
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m, _ = update(m, m.Init()())
	m, cmd := update(m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	if m.page != filesPage {
		t.Fatal("expected files page after 'f'")
	}
	if cmd == nil {
		t.Fatal("expected files command")
	}
	m, _ = update(m, cmd())
	return m
}

func TestOpenFilesListsWorkdir(t *testing.T) {
	m := openFiles(t, newFilesClient())
	got := ansi.Strip(m.View().Content)
	for _, want := range []string{"/app", "Name    web", "static", "main.go", "2.0 KB", "open dir", "esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}
}

func TestOpenDir(t *testing.T) {
	m := openFiles(t, newFilesClient())
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // main.go
	if m, cmd := update(m, enter); cmd != nil || m.filesSession != 1 {
		t.Error("enter on a file must not fetch")
	}

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyUp}) // static
	m, cmd := update(m, enter)
	if cmd == nil {
		t.Fatal("expected files command")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "main.go") {
		t.Error("current listing must stay until the new one arrives")
	}
	m, _ = update(m, cmd())
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "/app/static") || !strings.Contains(got, "index.html") || strings.Contains(got, "main.go") {
		t.Errorf("expected the listing of /app/static: %q", got)
	}
	if f, _ := m.files.Selected(); f.Name != ".." {
		t.Errorf("selected = %q, want the cursor reset to ..", f.Name)
	}
}

func TestFilesError(t *testing.T) {
	cli := newFilesClient()
	m := openFiles(t, cli)

	cli.filesErr = errors.New("boom")
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // static
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = update(m, cmd())
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Error: boom") || !strings.Contains(got, "main.go") {
		t.Errorf("expected the error below the kept listing: %q", got)
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}
}

func TestFilesBack(t *testing.T) {
	m := openFiles(t, newFilesClient())

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // static
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.page != containersPage {
		t.Error("expected containers page after esc")
	}
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Containers") || !strings.Contains(got, "Total") {
		t.Errorf("containers pane and counts not shown: %q", got)
	}

	// The listing requested before esc arrives late and must be dropped.
	m, _ = update(m, cmd())
	if strings.Contains(ansi.Strip(m.files.View()), "index.html") {
		t.Error("stale listing was applied")
	}
}

func TestFilesKeyIgnoredOnFilesPage(t *testing.T) {
	m := openFiles(t, newFilesClient())
	session := m.filesSession
	m, _ = update(m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	if m.filesSession != session {
		t.Error("'f' on the files page must not start a new request")
	}
}

func TestTerminalKey(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m, _ = update(m, m.Init()())
	if _, cmd := update(m, tea.KeyPressMsg{Code: 't', Text: "t"}); cmd == nil {
		t.Error("expected terminal command on the containers page")
	}

	m, _ = openLogs(t, &fakeClient{shorts: shorts})
	if _, cmd := update(m, tea.KeyPressMsg{Code: 't', Text: "t"}); cmd != nil {
		t.Error("'t' on the logs page must not open a terminal")
	}
	m = openFiles(t, newFilesClient())
	if _, cmd := update(m, tea.KeyPressMsg{Code: 't', Text: "t"}); cmd != nil {
		t.Error("'t' on the files page must not open a terminal")
	}
}

func TestTerminalError(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m, _ = update(m, m.Init()())

	m, _ = update(m, terminalMsg{err: errors.New("no shell found")})
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Error: no shell found") || !strings.Contains(got, "nginx") {
		t.Errorf("expected the error below the containers: %q", got)
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if strings.Contains(m.View().Content, "no shell found") {
		t.Error("error not cleared by the next key")
	}
}

// TestTerminalCmdStopsInput checks that the input handed to the shell stops
// reading once the session ends, like the input copy in client.Terminal
// that outlives the shell.
func TestTerminalCmdStopsInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	read := make(chan string, 1)
	cli := &fakeClient{terminal: func(id string, in io.Reader, out io.Writer) error {
		if id != shorts[0].ID {
			return errors.New("unexpected container " + id)
		}
		go func() {
			b, _ := io.ReadAll(in) // returns once the input is canceled
			read <- string(b)
		}()
		_, err := io.WriteString(out, "$ ")
		return err
	}}

	var out strings.Builder
	c := &terminalCmd{cli: cli, id: shorts[0].ID}
	c.SetStdin(r)
	c.SetStdout(&out)
	if err := c.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() != "$ " {
		t.Errorf("output = %q, want the shell output", out.String())
	}

	if _, err := w.WriteString("x"); err != nil {
		t.Fatal(err)
	}
	if got := <-read; got != "" {
		t.Errorf("shell read %q after it exited", got)
	}
	b := make([]byte, 1)
	if _, err := r.Read(b); err != nil || b[0] != 'x' {
		t.Errorf("key after the session = %q, %v; want it left for the TUI", b, err)
	}
}
