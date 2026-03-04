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
	client         dcli.APIClient
	sshClient      io.Closer
	daemonHostname string
}

func New(hostname string) (*Client, error) {
	opts := []dcli.Opt{dcli.FromEnv, dcli.WithAPIVersionNegotiation()}

	client := &Client{}
	client.daemonHostname = "localhost"
	if hostname != "" {
		dialOpt, closer, err := getSSHDialContext(hostname)
		if err != nil {
			return nil, fmt.Errorf("get dial context: %w", err)
		}
		client.sshClient = closer
		opts = append(opts, dialOpt)
		client.daemonHostname = hostname
	}

	cli, err := dcli.NewClientWithOpts(opts...)
	if err != nil {
		err = errors.Join(err, client.sshClient.Close())
		return nil, fmt.Errorf("create api client: %w", err)
	}
	client.client = cli
	return client, nil
}

func (cli *Client) Close() error {
	err := cli.client.Close()
	if cli.sshClient != nil {
		err = errors.Join(err, cli.sshClient.Close())
	}
	return err
}

func getSSHDialContext(host string) (dcli.Opt, io.Closer, error) {
	sshCli, err := NewSSHClient(host)
	if err != nil {
		return nil, nil, fmt.Errorf("dial host %s: %w", host, err)
	}

	dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
		return sshCli.DialContext(ctx, "unix", "/var/run/docker.sock")
	}

	return dcli.WithDialContext(dialer), sshCli, nil
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

func (cli *Client) DaemonHostname() string {
	return cli.daemonHostname
}

func (cli *Client) DaemonVersion() string {
	return cli.client.ClientVersion()
}
