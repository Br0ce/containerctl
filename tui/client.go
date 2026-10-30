package tui

import (
	"context"
	"io"

	"github.com/Br0ce/containerctl/client"
	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/engine"
	"github.com/Br0ce/containerctl/file"
)

// Client is the subset of client.Client the TUI depends on.
type Client interface {
	AllShorts(ctx context.Context) ([]container.Short, error)
	Logs(ctx context.Context, id string) (client.LogSeq, context.CancelFunc)
	FilesIn(ctx context.Context, root file.Info) ([]file.Info, error)
	Details(ctx context.Context, id string) (container.Details, error)
	Stats(ctx context.Context, id string) client.StatsSeq
	Terminal(ctx context.Context, id string, in io.Reader, out io.Writer) error
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string) error
	PauseContainer(ctx context.Context, id string) error
	UnpauseContainer(ctx context.Context, id string) error
	Engine(ctx context.Context) (engine.Info, error)
	DaemonHost() string
	DaemonVersion() string
}
