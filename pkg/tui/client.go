package tui

import (
	"context"

	"github.com/Br0ce/containerctl/pkg/client"
	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/file"
)

// Client is the subset of client.Client the TUI depends on.
type Client interface {
	AllShorts(ctx context.Context) ([]container.Short, error)
	Logs(ctx context.Context, id string) (client.LogSeq, context.CancelFunc)
	FilesIn(ctx context.Context, root file.Info) ([]file.Info, error)
	DaemonHost() string
	DaemonVersion() string
}
