package client

import (
	"archive/tar"
	"io/fs"
	"testing"
	"time"

	"github.com/Br0ce/containerctl/file"
)

func TestListDir(t *testing.T) {
	mod := time.Date(2026, 5, 13, 4, 21, 0, 0, time.UTC)
	tr := archive(t,
		dir("bin/"),
		entry{hdr: tar.Header{Typeflag: tar.TypeReg, Name: "bin/[", Mode: 0o755, ModTime: mod}, body: "busybox"},
		// Hard links have no data of their own, like busybox applets.
		entry{hdr: tar.Header{Typeflag: tar.TypeLink, Name: "bin/ls", Linkname: "bin/[", Mode: 0o755, ModTime: mod}},
		entry{hdr: tar.Header{Typeflag: tar.TypeSymlink, Name: "bin/sh", Linkname: "busybox", Mode: 0o777, ModTime: mod}},
		entry{hdr: tar.Header{Typeflag: tar.TypeReg, Name: "bin/empty", Mode: 0o600, ModTime: mod}},
		dir("bin/sub/"),
		reg("bin/sub/deep", "not listed"),
	)

	got, err := listDir(tr, file.Info{Path: "/bin", ContainerID: "c1"})
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		name   string
		size   int64
		mode   fs.FileMode
		target string
	}{
		{"..", 0, 0, ""},
		{"[", 7, 0o755, ""},
		{"ls", 7, 0o755, ""},
		{"sh", 0, fs.ModeSymlink | 0o777, "busybox"},
		{"empty", 0, 0o600, ""},
		{"sub", 0, fs.ModeDir | 0o755, ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Name != w.name || g.Size != w.size || g.Mode != w.mode || g.LinkTarget != w.target {
			t.Errorf("entry %d = {%q %d %v %q}, want %+v", i, g.Name, g.Size, g.Mode, g.LinkTarget, w)
		}
	}
	if got[1].Path != "/bin/[" || !got[1].ModTime.Equal(mod) {
		t.Errorf("path or mtime not set: %+v", got[1])
	}
}
