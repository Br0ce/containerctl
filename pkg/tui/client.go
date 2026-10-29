package tui

import (
	"context"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/container"
)

// Client is the subset of client.Client the TUI depends on.
type Client interface {
	AllShorts(ctx context.Context) ([]container.Short, error)
	Logs(ctx context.Context, id string) (client.LogSeq, context.CancelFunc)
	DaemonHost() string
	DaemonVersion() string
}
