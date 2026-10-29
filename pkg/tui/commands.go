package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/pkg/container"
)

// requestTimeout bounds every client call made by a command.
const requestTimeout = 10 * time.Second

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
