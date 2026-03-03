package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"slices"

	dcont "github.com/docker/docker/api/types/container"
	dcli "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/Br0ce/containerctl/pkg/container"
)

type LogSeq = iter.Seq2[string, error]

type Client struct {
	client dcli.APIClient
}

func New() (*Client, error) {
	opts := []dcli.Opt{dcli.FromEnv, dcli.WithAPIVersionNegotiation()}
	cli, err := dcli.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{client: cli}, nil
}

func (cli *Client) Close() error {
	return cli.client.Close()
}

func (cli *Client) Shorts(ctx context.Context) ([]container.Short, error) {
	sums, err := cli.client.ContainerList(ctx, dcont.ListOptions{All: true, Latest: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	slices.SortFunc(sums, func(a, b dcont.Summary) int {
		if a.State == dcont.StateRunning && b.State != dcont.StateRunning {
			return -1
		}
		if a.State == b.State {
			return 0
		}
		return 1
	})

	var shorts []container.Short
	for _, sum := range sums {
		shorts = append(shorts, container.Short{
			ID:     sum.ID,
			Name:   sum.Names[0],
			Image:  sum.Image,
			Status: sum.Status,
			State:  container.StateFrom(sum.State),
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
			// StdCopy demultiplexes the stream, i.e. stdout and stderr.
			_, err := stdcopy.StdCopy(pw, pw, rc)
			pw.CloseWithError(err)
		}()

		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			if !yield(scanner.Text(), nil) {
				err = pr.Close()
				if err != nil {
					fmt.Printf("close log stream: %v\n", err)
				}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			yield("", err)
		}
	}
}

func (cli *Client) DaemonHostname() (string, error) {
	url, err := dcli.ParseHostURL(cli.client.DaemonHost())
	if err != nil {
		return "", fmt.Errorf("parse host URL: %w", err)
	}
	return url.Hostname(), nil
}

func (cli *Client) DaemonVersion() string {
	return cli.client.ClientVersion()
}
