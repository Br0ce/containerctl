package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"slices"

	dcont "github.com/docker/docker/api/types/container"
	dcli "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/Br0ce/containerctl/pkg/container"
)

type LogSeq = iter.Seq2[string, error]

type Client struct {
	client          dcli.APIClient
	sshClientCloser io.Closer
	daemonHost      string
}

func New(cOpts ...ClientOptions) (*Client, error) {
	cfg, err := NewConfig(cOpts...)
	if err != nil {
		return nil, fmt.Errorf("make client config: %w", err)
	}

	client := &Client{}
	client.daemonHost = cfg.host

	opts := []dcli.Opt{dcli.FromEnv, dcli.WithAPIVersionNegotiation()}
	if client.daemonHost != "localhost" {
		sshCli, err := NewSSHClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("dial host %s: %w", cfg.host, err)
		}

		dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
			return sshCli.DialContext(ctx, "unix", cfg.DockerHost())
		}

		client.sshClientCloser = sshCli
		opts = append(opts, dcli.WithDialContext(dialer))
	}

	cli, err := dcli.NewClientWithOpts(opts...)
	if err != nil {
		if client.sshClientCloser != nil {
			err = errors.Join(err, client.sshClientCloser.Close())
		}
		return nil, fmt.Errorf("create api client: %w", err)
	}
	client.client = cli

	return client, nil
}

func (cli *Client) Close() error {
	err := cli.client.Close()
	if cli.sshClientCloser != nil {
		err = errors.Join(err, cli.sshClientCloser.Close())
	}
	return err
}

func (cli *Client) AllShorts(ctx context.Context) ([]container.Short, error) {
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
			State:  sum.State,
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

func (cli *Client) StartContainer(ctx context.Context, id string) error {
	return cli.client.ContainerStart(ctx, id, dcont.StartOptions{})
}

func (cli *Client) StopContainer(ctx context.Context, id string) error {
	return cli.client.ContainerStop(ctx, id, dcont.StopOptions{})
}

func (cli *Client) PauseContainer(ctx context.Context, id string) error {
	return cli.client.ContainerPause(ctx, id)
}

func (cli *Client) UnpauseContainer(ctx context.Context, id string) error {
	return cli.client.ContainerUnpause(ctx, id)
}

func (cli *Client) DaemonHost() string {
	return cli.daemonHost
}

func (cli *Client) DaemonVersion() string {
	return cli.client.ClientVersion()
}
