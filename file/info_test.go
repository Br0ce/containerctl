package file

import (
	"io/fs"
	"testing"
	"time"
)

func TestPermString(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want string
	}{
		{0o644, "-rw-r--r--"},
		{fs.ModeDir | 0o755, "drwxr-xr-x"},
		{fs.ModeSymlink | 0o777, "lrwxrwxrwx"},
		{fs.ModeDevice | fs.ModeCharDevice | 0o666, "crw-rw-rw-"},
		{fs.ModeDevice | 0o660, "brw-rw----"},
		{fs.ModeNamedPipe | 0o600, "prw-------"},
		{fs.ModeSocket | 0o755, "srwxr-xr-x"},
		{fs.ModeSetuid | 0o4755, "-rwsr-xr-x"},
		{fs.ModeSetgid | 0o644, "-rw-r-Sr--"},
		{fs.ModeDir | fs.ModeSticky | 0o777, "drwxrwxrwt"},
		{fs.ModeDir | fs.ModeSticky | 0o776, "drwxrwxrwT"},
	}
	for _, tt := range tests {
		if got := (Info{Mode: tt.mode}).PermString(); got != tt.want {
			t.Errorf("PermString(%v) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestModTimeString(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		mod  time.Time
		want string
	}{
		{"zero", time.Time{}, ""},
		{"recent", time.Date(2026, 10, 2, 14, 3, 0, 0, time.UTC), "Oct  2 14:03"},
		{"older than six months", time.Date(2025, 12, 24, 8, 0, 0, 0, time.UTC), "Dec 24  2025"},
		{"future", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), "Jan  1  2027"},
	}
	for _, tt := range tests {
		if got := (Info{ModTime: tt.mod}).ModTimeString(now); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSizeString(t *testing.T) {
	tests := []struct {
		info Info
		want string
	}{
		{Info{Size: 0}, "0 B"},
		{Info{Size: 42}, "42 B"},
		{Info{Size: 3 * 1024 * 1024}, "3.0 MB"},
		{Info{IsDir: true, Size: 4096}, "-"},
		{Info{Mode: fs.ModeSymlink | 0o777}, "-"},
	}
	for _, tt := range tests {
		if got := tt.info.SizeString(); got != tt.want {
			t.Errorf("SizeString(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}
