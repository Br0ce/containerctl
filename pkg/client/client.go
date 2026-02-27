package client

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Client struct {
	client client.APIClient
}

func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{client: cli}, nil
}

func (c *Client) All(ctx context.Context) ([]string, error) {
	defer c.client.Close()
	containers, err := c.client.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var ids []string
	for _, container := range containers {
		ids = append(ids, container.ID)
	}
	return ids, nil
}
