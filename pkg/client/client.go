package client

import (
	"archive/tar"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/moby/moby/api/pkg/stdcopy"
	mcont "github.com/moby/moby/api/types/container"
	mcli "github.com/moby/moby/client"
	"golang.org/x/term"

	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/file"
)

type LogSeq = iter.Seq2[string, error]

type Client struct {
	client          mcli.APIClient
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

	opts := []mcli.Opt{mcli.FromEnv}
	if client.daemonHost != "localhost" {
		sshCli, err := NewSSHClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("dial host %s: %w", cfg.host, err)
		}

		dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
			return sshCli.DialContext(ctx, "unix", cfg.DockerHost())
		}

		client.sshClientCloser = sshCli
		opts = append(opts, mcli.WithDialContext(dialer))
	}

	cli, err := mcli.New(opts...)
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
	res, err := cli.client.ContainerList(ctx, mcli.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	sums := res.Items
	slices.SortFunc(sums, func(a, b mcont.Summary) int {
		if a.State == mcont.StateRunning && b.State != mcont.StateRunning {
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
			State:  string(sum.State),
		})
	}

	return shorts, nil
}

func (cli *Client) Logs(ctx context.Context, id string) (LogSeq, context.CancelFunc) {
	ctx, cancelFn := context.WithCancel(ctx)
	return func(yield func(string, error) bool) {
		defer cancelFn()
		opts := mcli.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		}
		rc, err := cli.client.ContainerLogs(ctx, id, opts)
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
				// nolint:gosec // G104: error intentionally ignored; closing the pipe will cause
				// the StdCopy goroutine to exit, which is the intended behavior when the consumer
				// stops consuming logs.
				_ = pr.Close()
				return
			}
		}
		if err := scanner.Err(); err != nil {
			yield("", err)
		}
	}, cancelFn
}

func (cli *Client) StartContainer(ctx context.Context, id string) error {
	_, err := cli.client.ContainerStart(ctx, id, mcli.ContainerStartOptions{})
	return err
}

func (cli *Client) StopContainer(ctx context.Context, id string) error {
	_, err := cli.client.ContainerStop(ctx, id, mcli.ContainerStopOptions{})
	return err
}

func (cli *Client) PauseContainer(ctx context.Context, id string) error {
	_, err := cli.client.ContainerPause(ctx, id, mcli.ContainerPauseOptions{})
	return err
}

func (cli *Client) UnpauseContainer(ctx context.Context, id string) error {
	_, err := cli.client.ContainerUnpause(ctx, id, mcli.ContainerUnpauseOptions{})
	return err
}

func (cli *Client) filesIn(ctx context.Context, root file.Info) ([]file.Info, error) {
	res, err := cli.client.CopyFromContainer(ctx, root.ContainerID, mcli.CopyFromContainerOptions{SourcePath: root.Path})
	if err != nil {
		return nil, fmt.Errorf("copy from container: %w", err)
	}
	defer res.Content.Close()

	// The Tar stream contains the working directory as its first entry, followed by its contents in “deep-first” order.
	tr := tar.NewReader(res.Content)

	// The first entry in the tar stream is the working directory, skip it.
	start, err := tr.Next()
	if err != nil {
		return nil, fmt.Errorf("skip first tar entry: %w", err)
	}
	var files []file.Info
	workdir := start.Name

	// If the working directory is not the root, we need to trim the prefix to get the correct paths for the children.
	if workdir != "/" {
		workdir = strings.TrimPrefix(filepath.Clean(start.Name), "/")

		// Add the parent directory entry first, so it appears at the top of the list.
		files = append(files, file.Info{
			Name:        "..",
			Path:        filepath.Dir(root.Path),
			IsDir:       true,
			ContainerID: root.ContainerID,
			DisplayName: root.Path,
		})
	}

	for {
		entry, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}

		// Only include immediate children of the working directory.
		dir := filepath.Dir(filepath.Clean(entry.Name))
		if workdir != dir {
			continue
		}

		fi := entry.FileInfo()
		files = append(files, file.Info{
			Name:        fi.Name(),
			Path:        filepath.Join(root.Path, fi.Name()),
			IsDir:       fi.IsDir(),
			ContainerID: root.ContainerID,
			DisplayName: root.Path,
			Size:        fi.Size(),
		})
	}
}

// FilesIn returns the files in the given directory inside the container.
// If the Path field of the root is empty, it defaults to the working directory of the container,
// or "/" if the working directory is not set.
func (cli *Client) FilesIn(ctx context.Context, root file.Info) ([]file.Info, error) {
	if root.Path != "" {
		return cli.filesIn(ctx, root)
	}

	// Since path is empty, we try to get the working directory of the container and set it as root.Path.
	info, err := cli.client.ContainerInspect(ctx, root.ContainerID, mcli.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect container %s: %w", root.ContainerID, err)
	}
	if info.Container.Config != nil {
		root.Path = info.Container.Config.WorkingDir
	}

	// If path is still not set, default to "/".
	if root.Path == "" {
		root.Path = "/"
	}

	return cli.filesIn(ctx, root)
}

func (cli *Client) findShell(ctx context.Context, id string) (string, error) {
	for _, shell := range []string{"/bin/sh", "/bin/bash", "/bin/ash"} {
		_, err := cli.client.ContainerStatPath(ctx, id, mcli.ContainerStatPathOptions{Path: shell})
		if err == nil {
			return shell, nil
		}
	}
	return "", fmt.Errorf("no shell found in container %s: tried /bin/sh, /bin/bash, /bin/ash", id)
}

// Terminal opens an interactive shell session inside the container identified by id.
// It creates a Docker exec process, attaches stdin/stdout/stderr with a PTY, and
// bridges in/out to the exec connection. The call blocks until the shell exits.
func (cli *Client) Terminal(ctx context.Context, id string, in io.Reader, out io.Writer) error {
	shell, err := cli.findShell(ctx, id)
	if err != nil {
		return err
	}

	// Seed the initial PTY dimensions from the current terminal; or fallback to 80x24
	cols, rows, err := term.GetSize(int(os.Stdout.Fd())) //nolint:gosec // G115: os.Stdout.Fd() returns a small non-negative file descriptor that always fits in int
	if err != nil {
		cols, rows = 80, 24
	}

	opts := mcli.ExecCreateOptions{
		Cmd:          []string{shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		ConsoleSize:  mcli.ConsoleSize{Height: uint(rows), Width: uint(cols)},
	}
	execID, err := cli.client.ExecCreate(ctx, id, opts)
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}

	// Attach to the exec instance to obtain a bidirectional connection to the shell.
	resp, err := cli.client.ExecAttach(ctx, execID.ID, mcli.ExecAttachOptions{
		TTY: true,
	})
	if err != nil {
		return fmt.Errorf("attach to exec: %w", err)
	}
	defer resp.Close()

	// Switch the local terminal to raw mode so keystrokes are forwarded as-is
	// to the shell rather than being processed by the local line discipline.
	//nolint:gosec // G115: os.Stdin.Fd() returns a small non-negative file descriptor that always fits in int
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:gosec // G115: same as above

	resizeCtx, resizeCancel := context.WithCancel(ctx)
	defer resizeCancel()

	// Listen for SIGWINCH and forward terminal resize events to the container's PTY.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	go func() {
		for {
			select {
			case <-sigCh:
				//nolint:gosec // G115: os.Stdout.Fd() returns a small non-negative file descriptor that always fits in int
				w, h, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil {
					// Terminal size unavailable; skip this resize event and wait for the next SIGWINCH.
					continue
				}
				_, err = cli.client.ExecResize(resizeCtx, execID.ID, mcli.ExecResizeOptions{
					//nolint:gosec // G115: terminal width returned by term.GetSize is always positive, safe to convert to uint
					Width: uint(w),
					//nolint:gosec // G115: terminal height returned by term.GetSize is always positive, safe to convert to uint
					Height: uint(h),
				})
				if err != nil {
					// Resize is best-effort if we don't want to tearing down the session.
					continue
				}
			case <-resizeCtx.Done():
				return
			}
		}
	}()

	// Pump stdin into the exec connection in a separate goroutine so it does not
	// block the main goroutine that drains stdout/stderr below.
	go func() {
		//nolint:gosec // G104: error intentionally ignored; copy ends when the shell exits or the connection closes,
		// both are expected termination paths
		io.Copy(resp.Conn, in)
		//nolint:gosec // G104: error intentionally ignored
		resp.CloseWrite()
	}()

	// Stream shell output to out; returns when the shell exits and the connection is closed.
	_, err = io.Copy(out, resp.Reader)
	return err
}

func (cli *Client) DaemonHost() string {
	return cli.daemonHost
}

func (cli *Client) DaemonVersion() string {
	return cli.client.ClientVersion()
}
