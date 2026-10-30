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

func TestLocalDockerHost(t *testing.T) {
	tests := []struct {
		host, want string
	}{
		{"/var/run/docker.sock", "unix:///var/run/docker.sock"},
		{"unix:///run/user/1000/podman/podman.sock", "unix:///run/user/1000/podman/podman.sock"},
		{"tcp://127.0.0.1:2375", "tcp://127.0.0.1:2375"},
	}
	for _, tt := range tests {
		if got := localDockerHost(tt.host); got != tt.want {
			t.Errorf("localDockerHost(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestRemoteSocket(t *testing.T) {
	tests := []struct {
		host, want string
	}{
		{"", "/var/run/docker.sock"},
		{"/run/podman/podman.sock", "/run/podman/podman.sock"},
		{"unix:///run/podman/podman.sock", "/run/podman/podman.sock"},
	}
	for _, tt := range tests {
		if got := remoteSocket(tt.host); got != tt.want {
			t.Errorf("remoteSocket(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}
