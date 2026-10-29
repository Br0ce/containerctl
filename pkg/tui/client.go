package tui

import (
	"context"

	"github.com/Br0ce/containerctl/pkg/container"
)

// Client is the subset of client.Client the TUI depends on.
type Client interface {
	AllShorts(ctx context.Context) ([]container.Short, error)
	DaemonHost() string
	DaemonVersion() string
}
