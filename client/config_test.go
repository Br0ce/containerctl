package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteFromSSHConfig(t *testing.T) {
	dir := t.TempDir()
	content := "Host myalias\n  HostName real.example.com\n  Port 2222\n  User alice\n"
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenInRoot(dir, "config")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	cfg, err := NewConfig(WithHost("myalias"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.rewriteFromSSHConfig(file); err != nil {
		t.Fatal(err)
	}

	if got, want := cfg.Host(), "real.example.com"; got != want {
		t.Errorf("Host() = %q, want %q", got, want)
	}
	if got, want := cfg.Addr(), "real.example.com:2222"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
	if got, want := cfg.Username(), "alice"; got != want {
		t.Errorf("Username() = %q, want %q", got, want)
	}
}
