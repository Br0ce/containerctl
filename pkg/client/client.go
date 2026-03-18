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

	dcont "github.com/docker/docker/api/types/container"
	dcli "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/term"

	"github.com/Br0ce/containerctl/pkg/container"
	"github.com/Br0ce/containerctl/pkg/file"
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

func (cli *Client) filesIn(ctx context.Context, root file.Info) ([]file.Info, error) {
	rc, _, err := cli.client.CopyFromContainer(ctx, root.ContainerID, root.Path)
	if err != nil {
		return nil, fmt.Errorf("copy from container: %w", err)
	}
	defer rc.Close()

	// The Tar stream contains the working directory as its first entry, followed by its contents in “deep-first” order.
	tr := tar.NewReader(rc)

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
	info, err := cli.client.ContainerInspect(ctx, root.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container %s: %w", root.ContainerID, err)
	}
	if info.Config != nil {
		root.Path = info.Config.WorkingDir
	}

	// If path is still not set, default to "/".
	if root.Path == "" {
		root.Path = "/"
	}

	return cli.filesIn(ctx, root)
}

func (cli *Client) findShell(ctx context.Context, id string) (string, error) {
	for _, shell := range []string{"/bin/sh", "/bin/bash", "/bin/ash"} {
		_, err := cli.client.ContainerStatPath(ctx, id, shell)
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

	opts := dcont.ExecOptions{
		Cmd:          []string{shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		ConsoleSize:  &[2]uint{uint(rows), uint(cols)},
	}
	execID, err := cli.client.ContainerExecCreate(ctx, id, opts)
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}

	// Attach to the exec instance to obtain a bidirectional connection to the shell.
	resp, err := cli.client.ContainerExecAttach(ctx, execID.ID, dcont.ExecStartOptions{
		Tty: true,
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
				err = cli.client.ContainerExecResize(resizeCtx, execID.ID, dcont.ResizeOptions{
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
