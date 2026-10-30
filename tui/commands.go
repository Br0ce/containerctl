package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/muesli/cancelreader"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/engine"
	"github.com/Br0ce/containerctl/file"
)

const (
	// requestTimeout bounds every client call made by a command, except
	// container actions.
	requestTimeout = 10 * time.Second
	// actionTimeout bounds a container action. It exceeds the 10 second grace
	// period Docker gives a container to stop before killing it.
	actionTimeout = 30 * time.Second
	// refreshInterval is the pause between the end of one container fetch
	// and the start of the next.
	refreshInterval = 3 * time.Second
	// statusDuration is how long a message stays on the status line.
	statusDuration = 5 * time.Second
)

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

type containersMsg struct {
	shorts []container.Short
	err    error
}

func fetchContainers(cli Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		shorts, err := cli.AllShorts(ctx)
		return containersMsg{shorts: shorts, err: err}
	}
}

// engineMsg carries the Docker host's information, fetched once at startup.
type engineMsg struct {
	info engine.Info
	err  error
}

func fetchEngine(cli Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		info, err := cli.Engine(ctx)
		return engineMsg{info: info, err: err}
	}
}

// filesMsg carries the listing of directory root. session identifies the
// request, so that listings arriving after the user moved on are dropped.
type filesMsg struct {
	session int
	root    file.Info
	files   []file.Info
	err     error
}

func fetchFiles(cli Client, session int, root file.Info) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		files, err := cli.FilesIn(ctx, root)
		return filesMsg{session: session, root: root, files: files, err: err}
	}
}

// detailsMsg carries the inspect information of a container. session
// identifies the request, so that details arriving after the user moved on
// are dropped.
type detailsMsg struct {
	session int
	details container.Details
	err     error
}

func fetchDetails(cli Client, session int, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		details, err := cli.Details(ctx, id)
		return detailsMsg{session: session, details: details, err: err}
	}
}

// actionMsg reports the end of a container action such as start or stop.
// verb and name describe the action for the status line.
type actionMsg struct {
	verb string
	name string
	err  error
}

// containerAction runs action on container s in the background. The
// container list picks up the new state on its next refresh.
func containerAction(verb string, action func(ctx context.Context, id string) error, s container.Short) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()

		return actionMsg{verb: verb, name: s.Name, err: action(ctx, s.ID)}
	}
}

// actionStatus renders a failed action as a short line, e.g.
// "stop web: permission denied".
func actionStatus(msg actionMsg) string {
	reason := msg.err.Error()
	if errors.Is(msg.err, context.DeadlineExceeded) {
		reason = "timed out"
	}
	reason = strings.TrimPrefix(reason, "Error response from daemon: ")
	return msg.verb + " " + msg.name + ": " + reason
}

// clearStatusMsg hides the status line, unless a newer message replaced the
// one it was scheduled for.
type clearStatusMsg struct {
	session int
}

func clearStatus(session int) tea.Cmd {
	return tea.Tick(statusDuration, func(time.Time) tea.Msg { return clearStatusMsg{session: session} })
}

// terminalMsg reports the end of a shell session.
type terminalMsg struct {
	err error
}

// openTerminal hands the terminal to a shell in container id. The program is
// paused until the shell exits.
func openTerminal(cli Client, id string) tea.Cmd {
	return tea.Exec(&terminalCmd{cli: cli, id: id}, func(err error) tea.Msg {
		return terminalMsg{err: err}
	})
}

// terminalCmd runs client.Terminal as a tea.ExecCommand.
type terminalCmd struct {
	cli Client
	id  string
	in  io.Reader
	out io.Writer
}

func (c *terminalCmd) SetStdin(r io.Reader)  { c.in = r }
func (c *terminalCmd) SetStdout(w io.Writer) { c.out = w }

// SetStderr is a no-op: the shell runs in a TTY, which merges stderr into
// stdout.
func (c *terminalCmd) SetStderr(io.Writer) {}

// Run blocks until the shell exits. client.Terminal keeps reading its input
// after that, which would swallow the first key pressed back in the TUI, so
// the input is canceled once Run returns.
func (c *terminalCmd) Run() error {
	cr, err := cancelreader.NewReader(c.in)
	if err != nil {
		return fmt.Errorf("terminal input: %w", err)
	}
	in := &stdin{CancelReader: cr}
	defer in.stop()
	return c.cli.Terminal(context.Background(), c.id, in, c.out)
}

// stdin closes a CancelReader only once no read is in flight anymore, so a
// read still waiting on the input never touches closed file descriptors.
type stdin struct {
	cancelreader.CancelReader
	mu sync.RWMutex
}

func (s *stdin) Read(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CancelReader.Read(p)
}

// stop cancels the input and closes it after the pending read, which
// returns right away once canceled. Later reads fail without blocking.
func (s *stdin) stop() {
	s.Cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.Close()
}

// maxLogBatch bounds the lines delivered by one logsMsg, so a burst of output
// is rendered in a few frames instead of one frame per line.
const maxLogBatch = 500

type logItem struct {
	line string
	err  error
}

// logsMsg carries the lines of a log stream that were ready when it was sent.
// done is set once the stream has ended, after err if it failed.
type logsMsg struct {
	session int
	stream  <-chan logItem
	lines   []string
	err     error
	done    bool
}

// streamLogs starts following the logs of container id in the background.
// The stream runs until it ends or the returned cancel func is called.
func streamLogs(cli Client, session int, id string) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	seq, _ := cli.Logs(ctx, id) // canceling ctx also stops the stream
	stream := make(chan logItem, maxLogBatch)
	go func() {
		defer close(stream)
		for line, err := range seq {
			select {
			case stream <- logItem{line: line, err: err}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return waitForLogs(session, stream), cancel
}

// waitForLogs blocks until the stream has lines and returns them along with
// any others that are ready, up to maxLogBatch.
func waitForLogs(session int, stream <-chan logItem) tea.Cmd {
	return func() tea.Msg {
		msg := logsMsg{session: session, stream: stream}
		for len(msg.lines) < maxLogBatch {
			var (
				item logItem
				ok   bool
			)
			if len(msg.lines) == 0 {
				item, ok = <-stream
			} else {
				select {
				case item, ok = <-stream:
				default:
					return msg
				}
			}
			if !ok {
				msg.done = true
				return msg
			}
			if item.err != nil {
				msg.err = item.err
				return msg
			}
			msg.lines = append(msg.lines, item.line)
		}
		return msg
	}
}

type statsItem struct {
	stats container.Stats
	err   error
}

// statsMsg carries the next sample of a stats stream. done is set once the
// stream has ended, after err if it failed.
type statsMsg struct {
	session int
	stream  <-chan statsItem
	stats   container.Stats
	err     error
	done    bool
}

// streamStats starts following the resource usage of container id in the
// background. The stream runs until it ends or the returned cancel func is
// called.
func streamStats(cli Client, session int, id string) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	seq := cli.Stats(ctx, id)
	// The daemon sends about one sample per second, so a small buffer is
	// enough to never hold up the stream.
	stream := make(chan statsItem, 1)
	go func() {
		defer close(stream)
		for s, err := range seq {
			select {
			case stream <- statsItem{stats: s, err: err}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return waitForStats(session, stream), cancel
}

// waitForStats blocks until the stream has the next sample.
func waitForStats(session int, stream <-chan statsItem) tea.Cmd {
	return func() tea.Msg {
		item, ok := <-stream
		return statsMsg{session: session, stream: stream, stats: item.stats, err: item.err, done: !ok}
	}
}
