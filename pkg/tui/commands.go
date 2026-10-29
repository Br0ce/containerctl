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
