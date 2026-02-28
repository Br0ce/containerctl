package client

import (
	"context"
	"fmt"

	dcont "github.com/docker/docker/api/types/container"
	dcli "github.com/docker/docker/client"

	"github.com/Br0ce/cctl/pkg/container"
)

type Client struct {
	client dcli.APIClient
}

func New() (*Client, error) {
	cli, err := dcli.NewClientWithOpts(dcli.FromEnv, dcli.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{client: cli}, nil
}

func (cli *Client) Shorts(ctx context.Context) ([]container.Short, error) {
	defer cli.client.Close()
	containers, err := cli.client.ContainerList(ctx, dcont.ListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var shorts []container.Short
	for _, cont := range containers {
		shorts = append(shorts, container.Short{
			ID:     cont.ID,
			Name:   cont.Names[0],
			Image:  cont.Image,
			Status: cont.Status,
		})
	}

	return shorts, nil
}
