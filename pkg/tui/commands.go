package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/pkg/container"
)

const (
	// requestTimeout bounds every client call made by a command.
	requestTimeout = 10 * time.Second
	// refreshInterval is the pause between the end of one container fetch
	// and the start of the next.
	refreshInterval = 3 * time.Second
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
