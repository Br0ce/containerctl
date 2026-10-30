package tui

import (
	"context"
	"errors"
	"image/color"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/client"
	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/engine"
	"github.com/Br0ce/containerctl/file"
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
	// copies records the copied files, e.g. "/app/main.go -> out.go".
	copies  []string
	copyErr error
	// terminal, if set, runs in place of the shell session.
	terminal func(id string, in io.Reader, out io.Writer) error
	// actions records the container actions in the order they ran, e.g.
	// "stop aaaaaaaaaaaaaaaa".
	actions    []string
	actionErr  error
	engine     engine.Info
	engineErr  error
	details    container.Details
	detailsErr error
	stats      []container.Stats
	statsErr   error
	statsCtx   context.Context
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

func (f *fakeClient) CopyFrom(ctx context.Context, src file.Info, dst string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		return "", errors.New("expected a deadline on the request context")
	}
	f.copies = append(f.copies, src.Path+" -> "+dst)
	if f.copyErr != nil {
		return "", f.copyErr
	}
	return "/home/user/" + dst, nil
}

func (f *fakeClient) Details(ctx context.Context, id string) (container.Details, error) {
	if _, ok := ctx.Deadline(); !ok {
		return container.Details{}, errors.New("expected a deadline on the request context")
	}
	if id != shorts[0].ID {
		return container.Details{}, errors.New("unexpected container " + id)
	}
	return f.details, f.detailsErr
}

func (f *fakeClient) Stats(ctx context.Context, _ string) client.StatsSeq {
	f.statsCtx = ctx
	return func(yield func(container.Stats, error) bool) {
		for _, st := range f.stats {
			if !yield(st, nil) {
				return
			}
		}
		if f.statsErr != nil {
			yield(container.Stats{}, f.statsErr)
		}
	}
}

func (f *fakeClient) Terminal(_ context.Context, id string, in io.Reader, out io.Writer) error {
	return f.terminal(id, in, out)
}

func (f *fakeClient) action(ctx context.Context, name, id string) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("expected a deadline on the request context")
	}
	f.actions = append(f.actions, name+" "+id)
	return f.actionErr
}

func (f *fakeClient) StartContainer(ctx context.Context, id string) error {
	return f.action(ctx, "start", id)
}

func (f *fakeClient) StopContainer(ctx context.Context, id string) error {
	return f.action(ctx, "stop", id)
}

func (f *fakeClient) RestartContainer(ctx context.Context, id string) error {
	return f.action(ctx, "restart", id)
}

func (f *fakeClient) RemoveContainer(ctx context.Context, id string) error {
	return f.action(ctx, "remove", id)
}

func (f *fakeClient) PauseContainer(ctx context.Context, id string) error {
	return f.action(ctx, "pause", id)
}

func (f *fakeClient) UnpauseContainer(ctx context.Context, id string) error {
	return f.action(ctx, "unpause", id)
}

func (f *fakeClient) Engine(ctx context.Context) (engine.Info, error) {
	if _, ok := ctx.Deadline(); !ok {
		return engine.Info{}, errors.New("expected a deadline on the request context")
	}
	return f.engine, f.engineErr
}

func (f *fakeClient) DaemonHost() string    { return "unix:///var/run/docker.sock" }
func (f *fakeClient) DaemonVersion() string { return "1.47" }

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// runInit applies the messages of the commands Init starts.
func runInit(m Model) Model {
	for _, cmd := range m.Init()().(tea.BatchMsg) {
		m, _ = update(m, cmd())
	}
	return m
}

func TestInitLoadsEngine(t *testing.T) {
	info := engine.Info{Hostname: "prod-01", Version: "29.0.1", APIVersion: "1.52"}
	m := newModel(&fakeClient{engine: info})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})
	m = runInit(m)
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "prod-01") || !strings.Contains(got, "29.0.1 (API 1.52)") {
		t.Errorf("header doesn't show the engine:\n%s", got)
	}

	m = newModel(&fakeClient{engineErr: errors.New("boom")})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})
	m = runInit(m)
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "Version: 1.47") || strings.Contains(got, "boom") {
		t.Errorf("header doesn't fall back to the client version:\n%s", got)
	}
}

func TestInitLoadsContainers(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})

	m = runInit(m)
	if m.err != nil {
		t.Fatalf("unexpected error: %v", m.err)
	}
	if s, ok := m.containers.Selected(); !ok || s.ID != shorts[0].ID {
		t.Errorf("selected = %+v, want %s", s, shorts[0].ID)
	}
}

func TestBackgroundColorStylesHeaders(t *testing.T) {
	// The bold column header for ID.
	const teal, blue, darkTeal = "38;2;45;212;191;1mID", "38;2;30;144;255;1mID", "38;2;15;118;110;1mID"
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})
	m = runInit(m)
	if got := m.View().Content; !strings.Contains(got, blue) || strings.Contains(got, teal) {
		t.Errorf("headers aren't blue before the background is known:\n%q", got)
	}

	m, _ = update(m, tea.BackgroundColorMsg{Color: color.Black})
	if got := m.View().Content; !strings.Contains(got, teal) {
		t.Errorf("headers aren't teal on a dark background:\n%q", got)
	}

	m, _ = update(m, tea.BackgroundColorMsg{Color: color.White})
	if got := m.View().Content; !strings.Contains(got, darkTeal) {
		t.Errorf("headers aren't dark teal on a light background:\n%q", got)
	}
}

func TestFetchError(t *testing.T) {
	m := newModel(&fakeClient{err: errors.New("boom")})
	m = runInit(m)
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
	m = runInit(m)

	got := m.View().Content
	if !strings.Contains(got, "restart") || strings.Contains(got, "go to start") {
		t.Fatalf("short help expected by default: %q", got)
	}
	if h := lipgloss.Height(m.footer()); h != 1 {
		t.Errorf("short help height = %d, want 1", h)
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	got = m.View().Content
	if !strings.Contains(got, "go to start") {
		t.Errorf("full help expected after '?': %q", got)
	}
	if h := lipgloss.Height(m.footer()); h != 2 {
		t.Errorf("full help height = %d, want 2", h)
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if strings.Contains(m.View().Content, "go to start") {
		t.Error("short help not shown after second '?'")
	}
}

// TestFooterFitsWidth checks that no help line is wider than the window, on
// every page and in both help modes.
func TestFooterFitsWidth(t *testing.T) {
	const width = 80
	for _, p := range []page{containersPage, logsPage, filesPage, inspectPage, statsPage} {
		for _, all := range []bool{false, true} {
			m := newModel(&fakeClient{shorts: shorts})
			m, _ = update(m, tea.WindowSizeMsg{Width: width, Height: 20})
			m.page = p
			m.help.ShowAll = all
			if w := lipgloss.Width(m.footer()); w > width {
				t.Errorf("page %d, full help %v: footer width = %d, want <= %d", p, all, w, width)
			}
		}
	}
}

// openLogs loads the containers and presses l on the selected one.
func openLogs(t *testing.T, cli *fakeClient) (Model, tea.Cmd) {
	t.Helper()
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)
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
	for _, want := range []string{"Name:    web", "Image:   nginx", "first line", "second line", "esc"} {
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
	if !strings.Contains(got, "Containers") || !strings.Contains(got, "Total") || strings.Contains(got, "Image:   nginx") {
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
	m = runInit(m)
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
	for _, want := range []string{"/app", "Name:    web", "static", "main.go", "2.0 KB", "open dir", "esc"} {
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

// typeText presses the keys for s.
func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _ = update(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestCopyFile(t *testing.T) {
	cli := newFilesClient()
	m := openFiles(t, cli)
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // main.go

	m, _ = update(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	if got := ansi.Strip(m.footer()); !strings.Contains(got, "Copy main.go to: main.go") {
		t.Fatalf("expected the destination prompt with the file name: %q", got)
	}
	if h := lipgloss.Height(m.View().Content); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	// Keys edit the destination instead of running their bindings.
	m = typeText(m, ".bak q")
	if m.page != filesPage {
		t.Fatal("keys in the prompt left the files page")
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected copy command")
	}
	if strings.Contains(ansi.Strip(m.footer()), "Copy main.go") {
		t.Error("prompt still shown after enter")
	}
	m, _ = update(m, cmd())
	if len(cli.copies) != 1 || cli.copies[0] != "/app/main.go -> main.go.bak" {
		t.Errorf("copies = %q", cli.copies)
	}
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "copied main.go to /home/user/main.go.bak") {
		t.Errorf("expected the result on the status line: %q", got)
	}
}

func TestCopyDir(t *testing.T) {
	cli := newFilesClient()
	m := openFiles(t, cli)
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // static
	m, _ = update(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected copy command")
	}
	_, _ = update(m, cmd())
	if len(cli.copies) != 1 || cli.copies[0] != "/app/static -> static" {
		t.Errorf("copies = %q", cli.copies)
	}
}

func TestCopyCanceled(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, {Code: 'c', Mod: tea.ModCtrl}} {
		cli := newFilesClient()
		m := openFiles(t, cli)
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
		m, _ = update(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
		m, cmd := update(m, k)
		if cmd != nil || m.page != filesPage {
			t.Errorf("%s: expected the prompt closed without a command or leaving the page", k)
		}
		if strings.Contains(ansi.Strip(m.footer()), "Copy") {
			t.Errorf("%s: prompt still shown", k)
		}
	}
}

func TestCopySkipsParent(t *testing.T) {
	m := openFiles(t, newFilesClient()) // .. is selected
	if m, _ := update(m, tea.KeyPressMsg{Code: 'c', Text: "c"}); m.copying.Path != "" {
		t.Error("expected no prompt for the parent entry")
	}
}

func TestCopyEmptyDestination(t *testing.T) {
	cli := newFilesClient()
	m := openFiles(t, cli)
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = update(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	m, _ = update(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}) // delete before cursor
	if _, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Error("expected no copy without a destination")
	}
}

func TestCopyError(t *testing.T) {
	cli := newFilesClient()
	cli.copyErr = errors.New("/home/user/main.go already exists")
	m := openFiles(t, cli)
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown}) // main.go
	m, _ = update(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = update(m, cmd())
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "copy main.go: /home/user/main.go already exists") {
		t.Errorf("expected the error on the status line: %q", got)
	}
}

func TestTerminalKey(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)
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
	m = runInit(m)

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

func TestActionKeys(t *testing.T) {
	for _, tc := range []struct {
		key  rune
		want string
	}{
		{'s', "start"},
		{'x', "stop"},
		{'p', "pause"},
		{'u', "unpause"},
		{'r', "restart"},
	} {
		cli := &fakeClient{shorts: shorts}
		m := newModel(cli)
		m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
		m = runInit(m)

		_, cmd := update(m, tea.KeyPressMsg{Code: tc.key, Text: string(tc.key)})
		if cmd == nil {
			t.Fatalf("expected action command for %q", tc.key)
		}
		msg, ok := cmd().(actionMsg)
		if !ok || msg.err != nil {
			t.Fatalf("%q: got %+v, want actionMsg without error", tc.key, msg)
		}
		if want := tc.want + " " + shorts[0].ID; len(cli.actions) != 1 || cli.actions[0] != want {
			t.Errorf("%q: actions = %v, want [%s]", tc.key, cli.actions, want)
		}
	}
}

func TestRemoveAsksFirst(t *testing.T) {
	cli := &fakeClient{shorts: shorts}
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)

	m, cmd := update(m, tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd != nil {
		t.Fatal("'d' must not remove before confirmation")
	}
	if got := m.View().Content; !strings.Contains(got, "Remove container web? (y/N)") {
		t.Errorf("expected the confirmation prompt in the footer: %q", got)
	}
	if h := lipgloss.Height(m.View().Content); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, cmd = update(m, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("expected remove command after 'y'")
	}
	if msg, ok := cmd().(actionMsg); !ok || msg.err != nil {
		t.Fatalf("got %+v, want actionMsg without error", msg)
	}
	if want := "remove " + shorts[0].ID; len(cli.actions) != 1 || cli.actions[0] != want {
		t.Errorf("actions = %v, want [%s]", cli.actions, want)
	}
	if strings.Contains(m.View().Content, "Remove container") {
		t.Error("prompt still shown after confirmation")
	}
}

func TestRemoveCanceled(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{
		{Code: 'n', Text: "n"},
		{Code: tea.KeyEscape},
		// Keys with a meaning of their own are swallowed by the prompt.
		{Code: 'q', Text: "q"},
		{Code: 'l', Text: "l"},
		{Code: 's', Text: "s"},
	} {
		cli := &fakeClient{shorts: shorts}
		m := newModel(cli)
		m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
		m = runInit(m)

		m, _ = update(m, tea.KeyPressMsg{Code: 'd', Text: "d"})
		m, cmd := update(m, k)
		if cmd != nil || len(cli.actions) != 0 {
			t.Errorf("%s: expected no command and no action, got actions %v", k, cli.actions)
		}
		if m.page != containersPage || strings.Contains(m.View().Content, "Remove container") {
			t.Errorf("%s: expected the containers page without prompt", k)
		}
	}
}

func TestActionKeysIgnoredOnOtherPages(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		t.Error("'x' on the logs page must not stop a container")
	}
	m = openFiles(t, newFilesClient())
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		t.Error("'x' on the files page must not stop a container")
	}
}

func TestActionKeyWithoutContainers(t *testing.T) {
	m := newModel(&fakeClient{})
	m = runInit(m)
	if _, cmd := update(m, tea.KeyPressMsg{Code: 's', Text: "s"}); cmd != nil {
		t.Error("expected no command without a selected container")
	}
}

func TestActionError(t *testing.T) {
	cli := &fakeClient{shorts: shorts, actionErr: errors.New("Error response from daemon: permission denied")}
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)

	_, cmd := update(m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	m, _ = update(m, cmd())
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "stop web: permission denied") || !strings.Contains(got, "nginx") {
		t.Errorf("expected the error on the status line below the containers: %q", got)
	}
	if h := lipgloss.Height(m.View().Content); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	// Navigating keeps the message; only its timer or the next action clears
	// it.
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if !strings.Contains(m.View().Content, "permission denied") {
		t.Error("error cleared by a navigation key")
	}
}

func TestActionErrorTimeout(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)

	m, _ = update(m, actionMsg{verb: "stop", name: "web", err: context.DeadlineExceeded})
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "stop web: timed out") {
		t.Errorf("expected a timeout on the status line: %q", got)
	}
}

func TestActionErrorOnOtherPage(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})
	m, _ = update(m, actionMsg{verb: "stop", name: "web", err: errors.New("permission denied")})
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "stop web: permission denied") {
		t.Errorf("expected the error on the logs page: %q", got)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "stop web: permission denied") {
		t.Errorf("error lost when going back to the containers: %q", got)
	}
}

func TestActionErrorFitsWidth(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m = runInit(m)

	m, _ = update(m, actionMsg{verb: "stop", name: "web", err: errors.New(strings.Repeat("very long reason ", 10))})
	for _, l := range strings.Split(m.View().Content, "\n") {
		if w := lipgloss.Width(l); w > 40 {
			t.Errorf("line width = %d, want <= 40: %q", w, l)
		}
	}
	if h := lipgloss.Height(m.View().Content); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}
}

func TestActionErrorClears(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)

	// The timers themselves aren't run, to keep the test fast; their
	// messages carry the session of the message they were scheduled for.
	m, cmd := update(m, actionMsg{verb: "stop", name: "web", err: errors.New("first")})
	if cmd == nil {
		t.Fatal("expected a timer to clear the status line")
	}
	first := clearStatusMsg{session: m.statusSession}
	m, _ = update(m, actionMsg{verb: "start", name: "web", err: errors.New("second")})
	second := clearStatusMsg{session: m.statusSession}

	// The timer of the first message must not clear the second one.
	m, _ = update(m, first)
	if !strings.Contains(m.View().Content, "second") {
		t.Error("newer message cleared by the timer of an older one")
	}
	m, _ = update(m, second)
	if strings.Contains(m.View().Content, "second") {
		t.Error("message not cleared by its timer")
	}
}

func TestActionClearsStatus(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)

	m, _ = update(m, actionMsg{verb: "stop", name: "web", err: errors.New("permission denied")})
	m, _ = update(m, tea.KeyPressMsg{Code: 's', Text: "s"})
	if strings.Contains(m.View().Content, "permission denied") {
		t.Error("status line not cleared by the next action")
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

// openInspect loads the containers, presses i on the selected one and
// delivers the details.
func openInspect(t *testing.T, cli *fakeClient) Model {
	t.Helper()
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)
	m, cmd := update(m, tea.KeyPressMsg{Code: 'i', Text: "i"})
	if m.page != inspectPage {
		t.Fatal("expected inspect page after 'i'")
	}
	if cmd == nil {
		t.Fatal("expected details command")
	}
	m, _ = update(m, cmd())
	return m
}

func newInspectClient() *fakeClient {
	return &fakeClient{shorts: shorts, details: container.Details{
		ID:    shorts[0].ID,
		Name:  "web",
		State: "running",
		Env:   []string{"DB_PASSWORD=hunter2"},
	}}
}

func TestOpenInspect(t *testing.T) {
	m := openInspect(t, newInspectClient())
	got := ansi.Strip(m.View().Content)
	for _, want := range []string{"Inspect", "Name:    web", "GENERAL", "running", "DB_PASSWORD=••••••", "refresh", "show env", "esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if strings.Contains(got, "hunter2") {
		t.Error("environment value shown before toggling")
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	m, _ = update(m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	got = ansi.Strip(m.View().Content)
	if !strings.Contains(got, "DB_PASSWORD=hunter2") || !strings.Contains(got, "hide env") {
		t.Errorf("expected the revealed environment: %q", got)
	}
}

func TestInspectRefresh(t *testing.T) {
	cli := newInspectClient()
	m := openInspect(t, cli)

	cli.details.State = "exited"
	m, cmd := update(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("expected details command after 'r'")
	}
	m, _ = update(m, cmd())
	if got := ansi.Strip(m.View().Content); !strings.Contains(got, "exited") {
		t.Errorf("expected the refreshed state: %q", got)
	}

	cli.detailsErr = errors.New("boom")
	m, cmd = update(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = update(m, cmd())
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Error: boom") || !strings.Contains(got, "exited") {
		t.Errorf("expected the error above the kept details: %q", got)
	}
}

func TestInspectBack(t *testing.T) {
	m := newModel(newInspectClient())
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)
	m, cmd := update(m, tea.KeyPressMsg{Code: 'i', Text: "i"})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.page != containersPage {
		t.Error("expected containers page after esc")
	}

	// The details requested before esc arrive late and must be dropped.
	m, _ = update(m, cmd())
	if strings.Contains(ansi.Strip(m.inspect.View()), "GENERAL") {
		t.Error("stale details were applied")
	}
}

func TestInspectKeysIgnoredOnOtherPages(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'i', Text: "i"}); cmd != nil {
		t.Error("'i' on the logs page must not inspect")
	}
	m = openInspect(t, newInspectClient())
	session := m.inspectSession
	if m, _ = update(m, tea.KeyPressMsg{Code: 'i', Text: "i"}); m.inspectSession != session {
		t.Error("'i' on the inspect page must not start a new request")
	}
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		t.Error("'x' on the inspect page must not stop a container")
	}
}

// openStats loads the containers and presses m on the selected one.
func openStats(t *testing.T, cli *fakeClient) (Model, tea.Cmd) {
	t.Helper()
	m := newModel(cli)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = runInit(m)
	m, cmd := update(m, tea.KeyPressMsg{Code: 'm', Text: "m"})
	if m.page != statsPage {
		t.Fatal("expected stats page after 'm'")
	}
	if cmd == nil {
		t.Fatal("expected stats command")
	}
	return m, cmd
}

var sample = container.Stats{
	Read:       time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC),
	CPUPercent: 12.5,
	OnlineCPUs: 2,
	MemUsage:   256 << 20,
	MemLimit:   1 << 30,
}

func TestOpenStatsStreamsSamples(t *testing.T) {
	m, cmd := openStats(t, &fakeClient{shorts: shorts, stats: []container.Stats{sample}})
	got := ansi.Strip(m.View().Content)
	for _, want := range []string{"Stats", "Name:    web", "Loading…", "esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}

	m, cmd = update(m, cmd())
	if cmd == nil {
		t.Fatal("expected the stream to be awaited again")
	}
	got = ansi.Strip(m.View().Content)
	for _, want := range []string{"12.5%", "256 MiB / 1 GiB"} {
		if !strings.Contains(got, want) {
			t.Errorf("view does not contain %q: %q", want, got)
		}
	}
	if h := lipgloss.Height(got); h != 20 {
		t.Errorf("view height = %d, want 20", h)
	}

	if msg := cmd().(statsMsg); !msg.done {
		t.Fatal("expected the stream to end")
	} else if _, cmd = update(m, msg); cmd != nil {
		t.Error("ended stream must not be awaited again")
	}
}

func TestStatsError(t *testing.T) {
	m, cmd := openStats(t, &fakeClient{shorts: shorts, stats: []container.Stats{sample}, statsErr: errors.New("boom")})
	for cmd != nil {
		m, cmd = update(m, cmd())
	}
	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "Error: boom") || !strings.Contains(got, "12.5%") {
		t.Errorf("expected the error above the last sample: %q", got)
	}
}

func TestStatsBack(t *testing.T) {
	cli := &fakeClient{shorts: shorts}
	m, _ := openStats(t, cli)

	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.page != containersPage {
		t.Error("expected containers page after esc")
	}
	if cli.statsCtx.Err() == nil {
		t.Error("expected stats stream to be canceled")
	}
}

func TestStaleStatsDropped(t *testing.T) {
	m, _ := openStats(t, &fakeClient{shorts: shorts})

	m, cmd := update(m, statsMsg{session: m.statsSession - 1, stats: sample})
	if cmd != nil {
		t.Error("stale stream must not be awaited again")
	}
	if strings.Contains(m.View().Content, "12.5%") {
		t.Error("view shows samples of a stale stream")
	}
}

func TestStatsKeysIgnoredOnOtherPages(t *testing.T) {
	m, _ := openLogs(t, &fakeClient{shorts: shorts})
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'm', Text: "m"}); cmd != nil {
		t.Error("'m' on the logs page must not open stats")
	}
	m, _ = openStats(t, &fakeClient{shorts: shorts})
	session := m.statsSession
	if m, _ = update(m, tea.KeyPressMsg{Code: 'm', Text: "m"}); m.statsSession != session {
		t.Error("'m' on the stats page must not start a new stream")
	}
	if _, cmd := update(m, tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		t.Error("'x' on the stats page must not stop a container")
	}
}
