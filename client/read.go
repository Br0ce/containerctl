package client

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	mcli "github.com/moby/moby/client"

	"github.com/Br0ce/containerctl/file"
)

// maxLinkHops bounds the symbolic links ReadFile follows, so a link cycle
// fails instead of looping.
const maxLinkHops = 8

// ReadFile returns up to limit bytes of the regular file f inside its
// container, following symbolic links. truncated reports whether the file
// is longer than limit.
func (cli *Client) ReadFile(ctx context.Context, f file.Info, limit int64) (data []byte, truncated bool, err error) {
	p := f.Path
	for range maxLinkHops {
		data, truncated, link, err := cli.readFile(ctx, f.ContainerID, p, limit)
		if err != nil || link == "" {
			return data, truncated, err
		}
		if !path.IsAbs(link) {
			link = path.Join(path.Dir(p), link)
		}
		p = link
	}
	return nil, false, fmt.Errorf("read %s: too many levels of symbolic links", f.Path)
}

// readFile reads the file at p, or returns the target of p if it is a
// symbolic link.
func (cli *Client) readFile(ctx context.Context, id, p string, limit int64) ([]byte, bool, string, error) {
	res, err := cli.client.CopyFromContainer(ctx, id, mcli.CopyFromContainerOptions{SourcePath: p})
	if err != nil {
		return nil, false, "", fmt.Errorf("copy from container: %w", err)
	}
	// Closing the body early stops the transfer of the rest of the file.
	defer res.Content.Close()

	data, truncated, link, err := readEntry(tar.NewReader(res.Content), limit)
	if err != nil {
		return nil, false, "", fmt.Errorf("read %s: %w", p, err)
	}
	return data, truncated, link, nil
}

// readEntry reads up to limit bytes of the first entry in tr, which
// CopyFromContainer returned for a single file. For a symbolic link it
// returns the link target instead.
func readEntry(tr *tar.Reader, limit int64) (data []byte, truncated bool, link string, err error) {
	hdr, err := tr.Next()
	if errors.Is(err, io.EOF) {
		return nil, false, "", errors.New("empty archive")
	}
	if err != nil {
		return nil, false, "", fmt.Errorf("read tar: %w", err)
	}
	switch hdr.Typeflag {
	case tar.TypeReg:
	case tar.TypeSymlink:
		return nil, false, hdr.Linkname, nil
	case tar.TypeDir:
		return nil, false, "", errors.New("is a directory")
	default:
		return nil, false, "", errors.New("not a regular file")
	}

	// One byte past the limit tells whether the file is longer.
	data, err = io.ReadAll(io.LimitReader(tr, limit+1))
	if err != nil {
		return nil, false, "", fmt.Errorf("read tar: %w", err)
	}
	if int64(len(data)) > limit {
		return data[:limit], true, "", nil
	}
	return data, false, "", nil
}
