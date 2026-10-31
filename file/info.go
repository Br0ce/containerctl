package file

import (
	"fmt"
	"io/fs"
	"time"
)

type Info struct {
	Name        string
	Path        string
	IsDir       bool
	ContainerID string
	DisplayName string
	Size        int64
	Mode        fs.FileMode
	ModTime     time.Time
	// LinkTarget is set for symbolic links.
	LinkTarget string
}

func (i Info) IsSymlink() bool {
	return i.Mode&fs.ModeSymlink != 0
}

func (i Info) SizeString() string {
	if i.IsDir || i.IsSymlink() {
		return "-"
	}

	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case i.Size >= gb:
		return fmt.Sprintf("%.1f GB", float64(i.Size)/gb)
	case i.Size >= mb:
		return fmt.Sprintf("%.1f MB", float64(i.Size)/mb)
	case i.Size >= kb:
		return fmt.Sprintf("%.1f KB", float64(i.Size)/kb)
	default:
		return fmt.Sprintf("%d B", i.Size)
	}
}

// PermString formats the mode like ls -l, e.g. "drwxr-xr-x" or "lrwxrwxrwx".
// fs.FileMode.String differs from ls for symlinks ("L") and the special bits.
func (i Info) PermString() string {
	b := []byte("----------")
	m := i.Mode
	switch {
	case m.IsDir():
		b[0] = 'd'
	case m&fs.ModeSymlink != 0:
		b[0] = 'l'
	case m&fs.ModeCharDevice != 0:
		b[0] = 'c'
	case m&fs.ModeDevice != 0:
		b[0] = 'b'
	case m&fs.ModeNamedPipe != 0:
		b[0] = 'p'
	case m&fs.ModeSocket != 0:
		b[0] = 's'
	}
	const rwx = "rwxrwxrwx"
	for j := range 9 {
		if m&(1<<(8-j)) != 0 {
			b[j+1] = rwx[j]
		}
	}
	special := func(pos int, set bool, lower, upper byte) {
		if !set {
			return
		}
		if b[pos] == 'x' {
			b[pos] = lower
		} else {
			b[pos] = upper
		}
	}
	special(3, m&fs.ModeSetuid != 0, 's', 'S')
	special(6, m&fs.ModeSetgid != 0, 's', 'S')
	special(9, m&fs.ModeSticky != 0, 't', 'T')
	return string(b)
}

// ModTimeString formats the modification time like ls -l: "Oct 12 14:03"
// for the last six months, "Oct 12  2025" otherwise.
func (i Info) ModTimeString(now time.Time) string {
	if i.ModTime.IsZero() {
		return ""
	}
	t := i.ModTime.In(now.Location())
	if t.After(now.AddDate(0, -6, 0)) && !t.After(now.Add(time.Hour)) {
		return t.Format("Jan _2 15:04")
	}
	return t.Format("Jan _2  2006")
}
