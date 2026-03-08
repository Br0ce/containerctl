package client

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	host         string
	dockerSocket string
	username     string
	identityFile string
	askPwd       bool
	sshDir       string
}

type ClientOptions func(*Config)

func WithHost(host string) ClientOptions {
	return func(cfg *Config) {
		cfg.host = host
	}
}

func WithDockerSocket(sock string) ClientOptions {
	return func(cfg *Config) {
		cfg.dockerSocket = sock
	}
}

func WithUsername(username string) ClientOptions {
	return func(cfg *Config) {
		cfg.username = username
	}
}

func WithIdentityFile(identityFile string) ClientOptions {
	return func(cfg *Config) {
		cfg.identityFile = identityFile
	}
}

func WithAskPassword(ask bool) ClientOptions {
	return func(cfg *Config) {
		cfg.askPwd = ask
	}
}

func MakeConfig(opts ...ClientOptions) (Config, error) {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	u, err := url.Parse("tcp://" + cfg.host)
	if err != nil {
		return Config{}, fmt.Errorf("parse host URL: %w", err)
	}
	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Host, "22")
	}
	if _, present := u.User.Password(); present {
		return Config{}, fmt.Errorf("username must be provided with --ask-password, not embedded in host")
	}
	cfg.host = u.Host

	curUser, err := user.Current()
	if err != nil {
		return Config{}, fmt.Errorf("get current user: %w", err)
	}
	cfg.sshDir = filepath.Join(curUser.HomeDir, ".ssh")

	if cfg.username != "" && u.User.Username() != "" {
		return Config{}, fmt.Errorf("username provided in both host and argument")
	}
	if cfg.username == "" {
		cfg.username = u.User.Username()
	}
	// If username is still empty, default to current user.
	if cfg.username == "" {
		cfg.username = curUser.Username
	}

	return *cfg, nil
}

func (cfg Config) Hostname() string {
	return cfg.host
}

func (cfg Config) Username() string {
	return cfg.username
}

func (cfg Config) AskPassword() bool {
	return cfg.askPwd
}

func (cfg Config) SSHDir() string {
	return cfg.sshDir
}

func (cfg Config) DockerSocket() string {
	return cfg.dockerSocket
}

// openIdentityFile tries to open the identity file from the given filename
// or if filename is empty, the default ones in the ~/.ssh directory.
// It returns an error if the file cannot be opened or if the filename is outside of the ~/.ssh directory.
func (cfg Config) OpenIdentityFile() (*os.File, error) {
	if cfg.identityFile != "" {
		return os.OpenInRoot(cfg.sshDir, filepath.Base(cfg.identityFile))
	}

	// No filename provided, try the defaults.
	var errs error
	for _, f := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		idFile, err := os.OpenInRoot(cfg.sshDir, f)
		if err == nil {
			return idFile, nil
		}
		errs = errors.Join(errs, err)
	}

	return nil, fmt.Errorf("open default identity file: %w", errs)
}
