package client

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	mcli "github.com/moby/moby/client"

	"github.com/Br0ce/containerctl/file"
)

// CopyFrom copies the file or directory src out of its container to dst on
// the local machine, like docker cp: if dst is an existing directory or ends
// with a separator, src is copied into it under its own name. Directories are
// copied recursively. Nothing is overwritten; if the target already exists,
// CopyFrom fails. It returns the absolute path that was written.
func (cli *Client) CopyFrom(ctx context.Context, src file.Info, dst string) (string, error) {
	dst, err := destination(src.Path, dst)
	if err != nil {
		return "", err
	}

	res, err := cli.client.CopyFromContainer(ctx, src.ContainerID, mcli.CopyFromContainerOptions{SourcePath: src.Path})
	if err != nil {
		return "", fmt.Errorf("copy from container: %w", err)
	}
	defer res.Content.Close()

	if err := extract(tar.NewReader(res.Content), dst); err != nil {
		return "", err
	}
	return dst, nil
}

// destination resolves dst for copying src: a leading ~ is expanded to the
// home directory, and an existing directory or a trailing separator gets the
// base name of src appended. It fails if the result already exists.
func destination(src, dst string) (string, error) {
	if dst == "" {
		return "", errors.New("no destination given")
	}
	if dst == "~" || strings.HasPrefix(dst, "~"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		dst = home + dst[1:]
	}

	if strings.HasSuffix(dst, string(filepath.Separator)) {
		dst = filepath.Join(dst, path.Base(src))
	} else if fi, err := os.Stat(dst); err == nil && fi.IsDir() {
		dst = filepath.Join(dst, path.Base(src))
	}

	dst, err := filepath.Abs(dst)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", dst, err)
	}
	if _, err := os.Lstat(dst); err == nil {
		return "", fmt.Errorf("%s already exists", dst)
	}
	return dst, nil
}

// extract writes the archive returned by CopyFromContainer to dst. The
// archive holds a single top-level entry, the copied file or directory, which
// is renamed to the base name of dst. All writes go through an os.Root at the
// parent of dst, so entries can't escape it, and nothing existing is
// replaced. If extracting fails, whatever was written is removed again.
func extract(tr *tar.Reader, dst string) (err error) {
	parent, err := os.OpenRoot(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}
	defer parent.Close()

	name := filepath.Base(dst)
	var top string // the top-level entry in the archive, empty before the first
	// created is set once the top-level entry exists, so a failed copy never
	// removes a file that was there before.
	var created bool
	defer func() {
		if err != nil && created {
			_ = parent.RemoveAll(name)
		}
	}()

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if top == "" {
				return errors.New("empty archive")
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		first := top == ""
		if first {
			top, _, _ = strings.Cut(strings.TrimPrefix(path.Clean(hdr.Name), "/"), "/")
		}
		target, err := entryTarget(hdr.Name, top, name)
		if err != nil {
			return err
		}
		err = writeEntry(parent, tr, hdr, target, top, name)
		if first && !errors.Is(err, fs.ErrExist) {
			created = true
		}
		if err != nil {
			return err
		}
	}
}

// entryTarget maps the archive path of an entry to its path below the parent
// of the destination, replacing the top-level entry top with name.
func entryTarget(entry, top, name string) (string, error) {
	head, rest, _ := strings.Cut(strings.TrimPrefix(path.Clean(entry), "/"), "/")
	if top != "" && head != top {
		return "", fmt.Errorf("unexpected tar entry %s", entry)
	}
	return filepath.Join(name, filepath.FromSlash(rest)), nil
}

// writeEntry creates the entry hdr at target. Devices, fifos and other
// special files are skipped.
func writeEntry(root *os.Root, r io.Reader, hdr *tar.Header, target, top, name string) error {
	perm := hdr.FileInfo().Mode().Perm()
	switch hdr.Typeflag {
	case tar.TypeDir:
		// The owner must be able to write the directory to fill it.
		if err := root.Mkdir(target, perm|0o700); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	case tar.TypeReg:
		f, err := root.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if _, err := io.Copy(f, r); err != nil {
			_ = f.Close()
			return fmt.Errorf("write %s: %w", hdr.Name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("write %s: %w", hdr.Name, err)
		}
	case tar.TypeSymlink:
		if err := root.Symlink(hdr.Linkname, target); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
	case tar.TypeLink:
		old, err := entryTarget(hdr.Linkname, top, name)
		if err != nil {
			return err
		}
		if err := root.Link(old, target); err != nil {
			return fmt.Errorf("create link: %w", err)
		}
	}
	return nil
}
