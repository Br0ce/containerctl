package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"

	dcont "github.com/docker/docker/api/types/container"
	dcli "github.com/docker/docker/client"

	"github.com/Br0ce/cctl/pkg/container"
)

type LogSeq = iter.Seq2[string, error]

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

func (cli *Client) Close() error {
	return cli.client.Close()
}

func (cli *Client) Shorts(ctx context.Context) ([]container.Short, error) {
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

func (cli *Client) Logs(ctx context.Context, id string) LogSeq {
	return func(yield func(string, error) bool) {
		rc, err := cli.client.ContainerLogs(ctx, id, dcont.LogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			yield("", fmt.Errorf("container logs: %w", err))
			return
		}
		defer rc.Close()

		pr, pw := io.Pipe()
		go func() {
			_, err := io.Copy(pw, rc)
			pw.CloseWithError(err)
		}()

		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			if !yield(scanner.Text(), nil) {
				pr.Close()
				return
			}
		}
		if err := scanner.Err(); err != nil {
			yield("", err)
		}
	}
}
