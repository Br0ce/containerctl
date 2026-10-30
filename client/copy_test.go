package client

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type entry struct {
	hdr  tar.Header
	body string
}

// archive builds a tar stream like the one CopyFromContainer returns.
func archive(t *testing.T, entries ...entry) *tar.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := e.hdr
		hdr.Size = int64(len(e.body))
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return tar.NewReader(&buf)
}

func reg(name, body string) entry {
	return entry{hdr: tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o644}, body: body}
}

func dir(name string) entry {
	return entry{hdr: tar.Header{Typeflag: tar.TypeDir, Name: name, Mode: 0o755}}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExtractFile(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "copy.log")
	if err := extract(archive(t, reg("app.log", "hello")), dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); got != "hello" {
		t.Errorf("content = %q, want hello", got)
	}
}

func TestExtractDir(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "etc-copy")
	tr := archive(t,
		dir("etc/"),
		reg("etc/hosts", "127.0.0.1 localhost"),
		dir("etc/ssl/"),
		reg("etc/ssl/cert.pem", "cert"),
		entry{hdr: tar.Header{Typeflag: tar.TypeSymlink, Name: "etc/localtime", Linkname: "/usr/share/zoneinfo/UTC"}},
		entry{hdr: tar.Header{Typeflag: tar.TypeLink, Name: "etc/hosts.bak", Linkname: "etc/hosts"}},
	)
	if err := extract(tr, dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "hosts")); got != "127.0.0.1 localhost" {
		t.Errorf("hosts = %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "ssl", "cert.pem")); got != "cert" {
		t.Errorf("cert.pem = %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "hosts.bak")); got != "127.0.0.1 localhost" {
		t.Errorf("hosts.bak = %q", got)
	}
	if got, err := os.Readlink(filepath.Join(dst, "localtime")); err != nil || got != "/usr/share/zoneinfo/UTC" {
		t.Errorf("localtime -> %q, %v", got, err)
	}
}

func TestExtractRefusesExisting(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(dst, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := extract(archive(t, reg("app.log", "new")), dst); err == nil {
		t.Error("expected an error for an existing destination")
	}
	if got := readFile(t, dst); got != "keep" {
		t.Errorf("existing file changed to %q", got)
	}
}

// TestExtractStaysInside checks that entries can't write outside the
// destination, neither by their name nor through a symlink.
func TestExtractStaysInside(t *testing.T) {
	tests := []struct {
		name    string
		entries []entry
	}{
		{"second top-level entry", []entry{dir("etc/"), reg("other", "x")}},
		{"dot dot", []entry{dir("etc/"), reg("etc/../../escaped", "x")}},
		{"through symlink", []entry{
			dir("etc/"),
			{hdr: tar.Header{Typeflag: tar.TypeSymlink, Name: "etc/link", Linkname: "../../"}},
			reg("etc/link/escaped", "x"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			parent := filepath.Join(base, "parent")
			if err := os.Mkdir(parent, 0o750); err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(parent, "etc")
			if err := extract(archive(t, tt.entries...), dst); err == nil {
				t.Error("expected an error")
			}
			for _, p := range []string{filepath.Join(base, "escaped"), filepath.Join(parent, "other"), dst} {
				if _, err := os.Lstat(p); err == nil {
					t.Errorf("%s exists after a failed copy", p)
				}
			}
		})
	}
}

func TestExtractEmpty(t *testing.T) {
	if err := extract(archive(t), filepath.Join(t.TempDir(), "x")); err == nil {
		t.Error("expected an error for an empty archive")
	}
}

func TestDestination(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	home := filepath.Join(tmp, "home")
	if err := os.Mkdir(home, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(tmp, "taken"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		dst     string
		want    string
		wantErr string
	}{
		{dst: "app.log", want: filepath.Join(tmp, "app.log")},
		{dst: "copy.log", want: filepath.Join(tmp, "copy.log")},
		{dst: ".", want: filepath.Join(tmp, "app.log")},
		{dst: "~", want: filepath.Join(home, "app.log")},
		{dst: "~/x.log", want: filepath.Join(home, "x.log")},
		{dst: "new/", want: filepath.Join(tmp, "new", "app.log")},
		{dst: "taken", wantErr: "already exists"},
		{dst: "", wantErr: "no destination"},
	}
	for _, tt := range tests {
		got, err := destination("/app/app.log", tt.dst)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("destination(%q) error = %v, want %q", tt.dst, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("destination(%q) = %q, %v, want %q", tt.dst, got, err, tt.want)
		}
	}
}
