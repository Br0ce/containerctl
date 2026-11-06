package client

import (
	"archive/tar"
	"testing"
)

func TestReadEntry(t *testing.T) {
	tests := []struct {
		name      string
		entry     entry
		limit     int64
		want      string
		truncated bool
		link      string
		wantErr   string
	}{
		{"whole file", reg("app.py", "print('hi')\n"), 100, "print('hi')\n", false, "", ""},
		{"exactly the limit", reg("a.txt", "abcd"), 4, "abcd", false, "", ""},
		{"truncated", reg("a.txt", "abcdef"), 4, "abcd", true, "", ""},
		{"empty", reg("empty", ""), 4, "", false, "", ""},
		{"symlink", entry{hdr: tar.Header{Typeflag: tar.TypeSymlink, Name: "sh", Linkname: "busybox"}}, 4, "", false, "busybox", ""},
		{"directory", dir("etc/"), 4, "", false, "", "is a directory"},
		{"fifo", entry{hdr: tar.Header{Typeflag: tar.TypeFifo, Name: "pipe"}}, 4, "", false, "", "not a regular file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, truncated, link, err := readEntry(archive(t, tt.entry), tt.limit)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want || truncated != tt.truncated || link != tt.link {
				t.Errorf("got (%q, %v, %q), want (%q, %v, %q)", data, truncated, link, tt.want, tt.truncated, tt.link)
			}
		})
	}
}

func TestReadEntryEmptyArchive(t *testing.T) {
	if _, _, _, err := readEntry(archive(t), 4); err == nil {
		t.Error("expected an error for an empty archive")
	}
}
