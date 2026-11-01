package sshconfig

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	// Relative includes resolve against home/.ssh.
	cfg, err := Decode(strings.NewReader("Include basic\n"), WithHome(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("web").Get("HostName"); got != "web01.example.com" {
		t.Errorf("Get(HostName) = %q, want %q", got, "web01.example.com")
	}
}

func TestDecodeDefaultHome(t *testing.T) {
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cfg, err := Decode(strings.NewReader("Include ~/.ssh/basic\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("web").Get("HostName"); got != "web01.example.com" {
		t.Errorf("Get(HostName) = %q, want %q", got, "web01.example.com")
	}
}

func TestDecodeErrors(t *testing.T) {
	sshDir := filepath.Join(home, ".ssh")
	tests := []struct {
		name, config, file string
		line               int
		want               string
	}{
		{"unterminated", "Host broken\n  HostName \"broken\n", "", 2, "unterminated quote"},
		{"missing argument", "# comment\nHost\n", "", 2, "host: missing argument"},
		// Errors in an included file report the line in that file.
		{"in include", "Include unterminated\n", filepath.Join(sshDir, "unterminated"), 2, "unterminated quote"},
		{"include loop", "Include loop\n", filepath.Join(sshDir, "loop"), 1, "too many nested includes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tt.config), WithHome(home))
			wantParseError(t, err, tt.file, tt.line, tt.want)
		})
	}
}

func TestDecodeSkipsUnreadableInclude(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read files without permission")
	}
	dir := t.TempDir()
	name := filepath.Join(dir, "secret.conf")
	if err := os.WriteFile(name, []byte("User secret\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	cfg, err := Decode(strings.NewReader("Include "+name+"\nUser visible\n"), WithHome(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("any").Get("User"); got != "visible" {
		t.Errorf("Get(User) = %q, want %q", got, "visible")
	}
}

func TestDecodeFile(t *testing.T) {
	tests := []struct {
		path, alias, key, want string
	}{
		// Relative includes resolve against the file's directory.
		{"testdata/etc/ssh/ssh_config", "x.corp.example.com", "User", "corp"},
		{filepath.Join(home, ".ssh", "include"), "work", "HostName", "work.example.com"},
	}
	for _, tt := range tests {
		cfg, err := DecodeFile(tt.path, WithHome(home))
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Lookup(tt.alias).Get(tt.key); got != tt.want {
			t.Errorf("DecodeFile(%q).Lookup(%q).Get(%q) = %q, want %q", tt.path, tt.alias, tt.key, got, tt.want)
		}
	}
}

func TestDecodeFileErrors(t *testing.T) {
	_, err := DecodeFile(filepath.Join(home, ".ssh", "missing"), WithHome(home))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("DecodeFile(missing) error = %v, want fs.ErrNotExist", err)
	}

	path := filepath.Join(home, ".ssh", "unterminated")
	_, err = DecodeFile(path, WithHome(home))
	wantParseError(t, err, path, 2, "unterminated quote")
}

func TestUserConfig(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.Mkdir(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"config":     "Include extra.conf\nHost web\n  HostName web.example.com\n",
		"extra.conf": "User alice\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(sshDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg, err := UserConfig(WithHome(dir))
	if err != nil {
		t.Fatal(err)
	}
	s := cfg.Lookup("web")
	if got := s.Get("HostName"); got != "web.example.com" {
		t.Errorf("Get(HostName) = %q, want %q", got, "web.example.com")
	}
	if got := s.Get("User"); got != "alice" {
		t.Errorf("Get(User) = %q, want %q", got, "alice")
	}
}

func TestUserConfigMissing(t *testing.T) {
	cfg, err := UserConfig(WithHome(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("web").Get("HostName"); got != "" {
		t.Errorf("Get(HostName) = %q, want empty", got)
	}
}

// SystemConfig is decodeOptional on systemConfigPath, both tested here without reading the
// machine's real system-wide config.
func TestSystemConfigPath(t *testing.T) {
	got := systemConfigPath()
	want := "/etc/ssh/ssh_config"
	if runtime.GOOS == "windows" {
		want = filepath.Join(os.Getenv("ProgramData"), "ssh", "ssh_config")
	}
	if got != want {
		t.Errorf("systemConfigPath() = %q, want %q", got, want)
	}
}

func TestDecodeOptional(t *testing.T) {
	tests := []struct {
		name, path, want string
	}{
		// Relative includes resolve against the file's directory.
		{"exists", "testdata/etc/ssh/ssh_config", "corp"},
		{"missing", filepath.Join(t.TempDir(), "ssh_config"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := newConfig([]Option{WithHome(home)})
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := c.decodeOptional(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Lookup("x.corp.example.com").Get("User"); got != tt.want {
				t.Errorf("Get(User) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithHome(t *testing.T) {
	cfg, err := Decode(strings.NewReader("Include ~/.ssh/basic\n"), WithHome(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("db").Get("User"); got != "bob" {
		t.Errorf("Get(User) = %q, want %q", got, "bob")
	}
}

func TestWithIncludeDir(t *testing.T) {
	cfg, err := Decode(strings.NewReader("Include ssh_config.d/*\n"), WithHome(home), WithIncludeDir("testdata/etc/ssh"))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("x.corp.example.com").Get("Port"); got != "2200" {
		t.Errorf("Get(Port) = %q, want %q", got, "2200")
	}

	// WithIncludeDir overrides the file's directory for DecodeFile.
	cfg, err = DecodeFile(filepath.Join(home, ".ssh", "include"), WithHome(home), WithIncludeDir(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("work").Get("HostName"); got != "" {
		t.Errorf("Get(HostName) = %q, want empty", got)
	}
}

func TestWithMaxIncludeDepth(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"config": "Include a.conf\n",
		"a.conf": "# a\nInclude b.conf\n",
		"b.conf": "User deep\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(dir, "config")

	cfg, err := DecodeFile(config, WithHome(home), WithMaxIncludeDepth(2))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Lookup("any").Get("User"); got != "deep" {
		t.Errorf("Get(User) = %q, want %q", got, "deep")
	}

	tests := []struct {
		depth int
		file  string
		line  int
	}{
		{1, filepath.Join(dir, "a.conf"), 2},
		{0, config, 1},
		{-1, config, 1},
	}
	for _, tt := range tests {
		_, err := DecodeFile(config, WithHome(home), WithMaxIncludeDepth(tt.depth))
		wantParseError(t, err, tt.file, tt.line, "too many nested includes")
	}
}
