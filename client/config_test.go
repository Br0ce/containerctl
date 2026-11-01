package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Br0ce/containerctl/sshconfig"
)

// newTestConfig returns the Config for opts with its ssh directory in a temp dir, where
// ~/.ssh/config has the given content.
func newTestConfig(t *testing.T, sshConfig string, opts ...ClientOptions) *Config {
	t.Helper()
	cfg, err := NewConfig(opts...)
	if err != nil {
		t.Fatal(err)
	}
	cfg.sshDir = filepath.Join(t.TempDir(), ".ssh")
	if err := os.Mkdir(cfg.sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.sshDir, "config"), []byte(sshConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestRewriteFromSSHConfig(t *testing.T) {
	content := "Host myalias\n  HostName real.example.com\n  Port 2222\n  User alice\n"
	cfg := newTestConfig(t, content, WithHost("myalias"))
	if err := cfg.rewriteFromSSHConfig(); err != nil {
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

func TestRewriteFromSSHConfigTokens(t *testing.T) {
	content := "Host prod-*\n  HostName %h.corp.example.com\n  User %u\n  IdentityFile ~/.ssh/%h_%r_key\n"
	cfg := newTestConfig(t, content, WithHost("prod-api"))
	cfg.local.User = "max"
	if err := cfg.rewriteFromSSHConfig(); err != nil {
		t.Fatal(err)
	}

	if got, want := cfg.Host(), "prod-api.corp.example.com"; got != want {
		t.Errorf("Host() = %q, want %q", got, want)
	}
	if got, want := cfg.Username(), "max"; got != want {
		t.Errorf("Username() = %q, want %q", got, want)
	}
	if got, want := cfg.identityFile, filepath.Join(cfg.sshDir, "prod-api.corp.example.com_max_key"); got != want {
		t.Errorf("identityFile = %q, want %q", got, want)
	}
}

func TestNewConfigHost(t *testing.T) {
	tests := []struct {
		host, wantHost, wantAddr, wantUser string
	}{
		{"web.example.com", "web.example.com", "web.example.com:22", ""},
		{"web.example.com:2222", "web.example.com", "web.example.com:2222", ""},
		{"alice@web.example.com:2222", "web.example.com", "web.example.com:2222", "alice"},
		{"[2001:db8::1]:2222", "2001:db8::1", "[2001:db8::1]:2222", ""},
	}
	for _, tt := range tests {
		cfg, err := NewConfig(WithHost(tt.host))
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Host(); got != tt.wantHost {
			t.Errorf("NewConfig(%q).Host() = %q, want %q", tt.host, got, tt.wantHost)
		}
		if got := cfg.Addr(); got != tt.wantAddr {
			t.Errorf("NewConfig(%q).Addr() = %q, want %q", tt.host, got, tt.wantAddr)
		}
		if got := cfg.Username(); tt.wantUser != "" && got != tt.wantUser {
			t.Errorf("NewConfig(%q).Username() = %q, want %q", tt.host, got, tt.wantUser)
		}
	}
}

func TestRewriteFromSSHConfigCommandLineWins(t *testing.T) {
	content := "Host myalias\n  HostName real.example.com\n  Port 2222\n  User alice\n  IdentityFile ~/.ssh/alice_key\n"

	tests := []struct {
		name               string
		opts               []ClientOptions
		wantAddr, wantUser string
		wantIdentityFile   string
	}{
		{"from config", []ClientOptions{WithHost("myalias")}, "real.example.com:2222", "alice", "~/.ssh/alice_key"},
		{"user and port in host", []ClientOptions{WithHost("bob@myalias:2022")}, "real.example.com:2022", "bob", "~/.ssh/alice_key"},
		{"flags", []ClientOptions{WithHost("myalias"), WithUsername("bob"), WithIdentityFile("bob_key")}, "real.example.com:2222", "bob", "bob_key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig(t, content, tt.opts...)
			if err := cfg.rewriteFromSSHConfig(); err != nil {
				t.Fatal(err)
			}
			if got := cfg.Addr(); got != tt.wantAddr {
				t.Errorf("Addr() = %q, want %q", got, tt.wantAddr)
			}
			if got := cfg.Username(); got != tt.wantUser {
				t.Errorf("Username() = %q, want %q", got, tt.wantUser)
			}
			wantIdentityFile := tt.wantIdentityFile
			if rest, ok := strings.CutPrefix(wantIdentityFile, "~/"); ok {
				wantIdentityFile = filepath.Join(filepath.Dir(cfg.sshDir), rest)
			}
			if got := cfg.identityFile; got != wantIdentityFile {
				t.Errorf("identityFile = %q, want %q", got, wantIdentityFile)
			}
		})
	}
}

func TestRewriteFromSSHConfigCommandLineTokens(t *testing.T) {
	content := "Host myalias\n  Port 2222\n  User alice\n  IdentityFile ~/.ssh/%r_%p_key\n"
	cfg := newTestConfig(t, content, WithHost("bob@myalias:2022"))
	if err := cfg.rewriteFromSSHConfig(); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.identityFile, filepath.Join(cfg.sshDir, "bob_2022_key"); got != want {
		t.Errorf("identityFile = %q, want %q", got, want)
	}
}

func TestRewriteFromSystemSSHConfig(t *testing.T) {
	userContent := "Host myalias\n  HostName real.example.com\n\nHost other\n  User alice\n"
	systemDir := t.TempDir()
	systemConfig := filepath.Join(systemDir, "ssh_config")
	if err := os.WriteFile(systemConfig, []byte("Include ssh_config.d/*.conf\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(systemDir, "ssh_config.d"), 0o700); err != nil {
		t.Fatal(err)
	}
	corp := "Host *\n  User corp\n  Port 2200\n"
	if err := os.WriteFile(filepath.Join(systemDir, "ssh_config.d", "50-corp.conf"), []byte(corp), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, host         string
		systemConfig       string // "" if disabled
		wantAddr, wantUser string // wantUser "" for the default user
	}{
		{"system disabled", "myalias", "", "real.example.com:22", ""},
		{"system fills in", "myalias", systemConfig, "real.example.com:2200", "corp"},
		{"user config wins", "other", systemConfig, "other:2200", "alice"},
		{"missing system config", "myalias", filepath.Join(systemDir, "missing"), "real.example.com:22", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig(t, userContent, WithHost(tt.host))
			cfg.systemSSHConfig = tt.systemConfig
			defaultUser := cfg.Username()
			if err := cfg.rewriteFromSSHConfig(); err != nil {
				t.Fatal(err)
			}
			if got := cfg.Addr(); got != tt.wantAddr {
				t.Errorf("Addr() = %q, want %q", got, tt.wantAddr)
			}
			wantUser := tt.wantUser
			if wantUser == "" {
				wantUser = defaultUser
			}
			if got := cfg.Username(); got != wantUser {
				t.Errorf("Username() = %q, want %q", got, wantUser)
			}
		})
	}
}

func TestReadSSHConfigs(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "dotfiles-config")
	if err := os.WriteFile(outside, []byte("Host myalias\n  HostName real.example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	unreadable := filepath.Join(dir, "unreadable")
	if err := os.WriteFile(unreadable, []byte("Host x\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		symlink      bool   // ~/.ssh/config is a symlink to a file outside ~/.ssh
		systemConfig string // "" if disabled
		wantHostName string // HostName looked up for myalias
		wantErr      string // "" if no error is expected
	}{
		{"missing configs are skipped", false, filepath.Join(dir, "missing"), "", ""},
		{"symlink outside ~/.ssh", true, "", "real.example.com", ""},
		{"unreadable system config", false, unreadable, "", "permission denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.systemConfig == unreadable && os.Geteuid() == 0 {
				t.Skip("root can read files without permission")
			}
			sshDir := filepath.Join(t.TempDir(), ".ssh")
			if err := os.Mkdir(sshDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if tt.symlink {
				if err := os.Symlink(outside, filepath.Join(sshDir, "config")); err != nil {
					t.Fatal(err)
				}
			}

			cfg, err := NewConfig(WithHost("myalias"))
			if err != nil {
				t.Fatal(err)
			}
			cfg.sshDir = sshDir
			cfg.systemSSHConfig = tt.systemConfig

			configs, err := cfg.readSSHConfigs()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("readSSHConfigs() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("readSSHConfigs() error = %v", err)
			}
			if got := sshconfig.Lookup("myalias", configs...).Get("HostName"); got != tt.wantHostName {
				t.Errorf("HostName = %q, want %q", got, tt.wantHostName)
			}
		})
	}
}

func TestWithSystemSSHConfig(t *testing.T) {
	for _, enable := range []bool{true, false} {
		cfg, err := NewConfig(WithHost("myalias"), WithSystemSSHConfig(enable))
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if enable {
			want = "/etc/ssh/ssh_config"
		}
		if cfg.systemSSHConfig != want {
			t.Errorf("WithSystemSSHConfig(%v): systemSSHConfig = %q, want %q", enable, cfg.systemSSHConfig, want)
		}
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
