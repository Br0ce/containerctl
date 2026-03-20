package client

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/kevinburke/ssh_config"
)

type Config struct {
	host         string
	dockerHost   string
	port         string
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

func WithDockerHost(host string) ClientOptions {
	return func(cfg *Config) {
		cfg.dockerHost = host
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

// NewSSHClient creates a new SSH client for the given optins.
//
// Note that the host, username, port and identity file can be rewritten during ssh client construction
// with the value provided in the ~/.ssh/config file.
//
// Use the method of the client to get the constructed Addr, Host, Port, Username, AskPassword, SSHDir
// and DockerSocket.
func NewConfig(opts ...ClientOptions) (*Config, error) {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	u, err := url.Parse("tcp://" + cfg.host)
	if err != nil {
		return nil, fmt.Errorf("parse host URL: %w", err)
	}
	if u.Port() != "" {
		cfg.port = u.Port()
	} else {
		// Default to port 22 if not specified.
		cfg.port = "22"
	}

	if _, present := u.User.Password(); present {
		return nil, fmt.Errorf("username must be provided with --ask-password, not embedded in host")
	}
	cfg.host = u.Host

	curUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	cfg.sshDir = filepath.Join(curUser.HomeDir, ".ssh")

	if cfg.username != "" && u.User.Username() != "" {
		return nil, fmt.Errorf("username provided in both host and argument")
	}
	if cfg.username == "" {
		cfg.username = u.User.Username()
	}
	// If username is still empty, default to current user.
	if cfg.username == "" {
		cfg.username = curUser.Username
	}

	return cfg, nil
}

func (cfg *Config) Addr() string {
	return net.JoinHostPort(cfg.host, cfg.port)
}

func (cfg *Config) Host() string {
	return cfg.host
}

func (cfg *Config) Port() string {
	return cfg.port
}

func (cfg *Config) Username() string {
	return cfg.username
}

func (cfg *Config) AskPassword() bool {
	return cfg.askPwd
}

func (cfg *Config) SSHDir() string {
	return cfg.sshDir
}

func (cfg *Config) DockerHost() string {
	return cfg.dockerHost
}

// openIdentityFile tries to open the identity file from the given filename
// or if filename is empty the default ones in the ~/.ssh directory are used,
// e.g., id_ed25519, id_rsa, and id_ecdsa.
//
// In either case, the file must be inside the ~/.ssh directory. Any file outside of it will not
// be accepted for security reasons.
//
// The caller is responsible for closing the file.
//
// It returns an error if the file cannot be opened or if the filename is outside of the ~/.ssh directory.
func (cfg *Config) openIdentityFile() (*os.File, error) {
	// If an identity file is provided, use it.
	if cfg.identityFile != "" {
		return os.OpenInRoot(cfg.sshDir, filepath.Base(cfg.identityFile))
	}

	// We have not found the identity file so far, try the defaults.
	var errs error
	for _, f := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		idFile, err := os.OpenInRoot(cfg.sshDir, f)
		if err == nil {
			return idFile, nil
		}
		errs = errors.Join(errs, err)
	}

	return nil, fmt.Errorf("open identity file: %w", errs)
}

// rewriteFromSSHConfig rewrites the host, username, port and identity file from the given ssh
// config file if present.
//
// It returns an error if the file cannot be read or parsed.
func (cfg *Config) rewriteFromSSHConfig(file *os.File) error {
	sshCfg, err := ssh_config.Decode(file)
	if err != nil {
		return fmt.Errorf("decode ssh config: %w", err)
	}

	host := cfg.host

	hostCfg, err := sshCfg.Get(host, "HostName")
	if err != nil {
		return fmt.Errorf("get ssh config for HostName %s: %w", cfg.Addr(), err)
	}
	if hostCfg != "" {
		cfg.host = hostCfg
	}

	portCfg, err := sshCfg.Get(host, "Port")
	if err != nil {
		return fmt.Errorf("get ssh config for Port %s: %w", cfg.Addr(), err)
	}
	if portCfg != "" {
		cfg.port = portCfg
	}

	userCfg, err := sshCfg.Get(host, "User")
	if err != nil {
		return fmt.Errorf("get ssh config for User %s: %w", cfg.Addr(), err)
	}
	if userCfg != "" {
		cfg.username = userCfg
	}

	if cfg.identityFile == "" {
		identityFileCfg, err := sshCfg.Get(host, "IdentityFile")
		if err != nil {
			return fmt.Errorf("get ssh config for IdentityFile %s: %w", cfg.Addr(), err)
		}
		cfg.identityFile = identityFileCfg
	}

	return nil
}
