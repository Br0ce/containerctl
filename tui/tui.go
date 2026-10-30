package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/client"
)

type Config struct {
	Host         string
	DockerHost   string
	Username     string
	IdentityFile string
	AskPassword  bool
}

type UI struct {
	cli *client.Client
}

// New connects the client and returns a UI instance.
// It does not start the TUI application; the caller must call Run() to do so.
// Note that the caller is responsible for calling Close() on the UI to clean up
// resources after Run() returns.
func New(cfg Config) (*UI, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	opts := []client.ClientOptions{client.WithHost(cfg.Host)}

	if cfg.DockerHost != "" {
		opts = append(opts, client.WithDockerHost(cfg.DockerHost))
	}

	if cfg.Username != "" {
		opts = append(opts, client.WithUsername(cfg.Username))
	}
	if cfg.IdentityFile != "" {
		opts = append(opts, client.WithIdentityFile(cfg.IdentityFile))
	}
	if cfg.AskPassword {
		opts = append(opts, client.WithAskPassword(true))
	}

	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &UI{cli: cli}, nil
}

// Run starts the TUI application and blocks until it exits.
func (ui *UI) Run(ctx context.Context) error {
	_, err := tea.NewProgram(newModel(ui.cli), tea.WithContext(ctx)).Run()
	return err
}

func (ui *UI) Close() error {
	return ui.cli.Close()
}
