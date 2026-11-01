package client

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/Br0ce/containerctl/sshconfig"
)

// systemSSHConfigPath is the system-wide ssh config, read after ~/.ssh/config if enabled.
const systemSSHConfigPath = "/etc/ssh/ssh_config"

type Config struct {
	host         string
	dockerHost   string
	port         string
	username     string
	identityFile string
	askPwd       bool
	sshDir       string
	localUser    string
	// systemSSHConfig is the path of the system-wide ssh config, or "" if it is not read.
	systemSSHConfig string
	// portSet and usernameSet are true if the port or username were given on the command
	// line. As in OpenSSH, they then take precedence over the ssh config.
	portSet     bool
	usernameSet bool
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

// WithSystemSSHConfig enables reading /etc/ssh/ssh_config after ~/.ssh/config.
func WithSystemSSHConfig(enable bool) ClientOptions {
	return func(cfg *Config) {
		cfg.systemSSHConfig = ""
		if enable {
			cfg.systemSSHConfig = systemSSHConfigPath
		}
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
		cfg.portSet = true
	} else {
		// Default to port 22 if not specified.
		cfg.port = "22"
	}

	if _, present := u.User.Password(); present {
		return nil, fmt.Errorf("username must be provided with --ask-password, not embedded in host")
	}
	cfg.host = u.Hostname()

	curUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	cfg.sshDir = filepath.Join(curUser.HomeDir, ".ssh")
	cfg.localUser = curUser.Username

	if cfg.username != "" && u.User.Username() != "" {
		return nil, fmt.Errorf("username provided in both host and argument")
	}
	if cfg.username == "" {
		cfg.username = u.User.Username()
	}
	cfg.usernameSet = cfg.username != ""
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

// readSSHConfigs reads ~/.ssh/config and, if enabled, the system-wide ssh config, in the
// order OpenSSH reads them. A config that does not exist is skipped.
//
// It returns an error if a config exists but cannot be opened or parsed, e.g. if
// ~/.ssh/config is a symlink outside ~/.ssh.
func (cfg *Config) readSSHConfigs() ([]*sshconfig.Config, error) {
	home := filepath.Dir(cfg.sshDir)
	var configs []*sshconfig.Config

	userPath := filepath.Join(cfg.sshDir, "config")
	userFile, err := os.OpenInRoot(cfg.sshDir, "config")
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		if info, lerr := os.Lstat(userPath); lerr == nil && info.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("open %s: %w (a symlink outside ~/.ssh is not supported)", userPath, err)
		}
		return nil, fmt.Errorf("open %s: %w", userPath, err)
	default:
		defer userFile.Close()
		userCfg, err := sshconfig.Decode(userFile, sshconfig.WithHome(home))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", userPath, err)
		}
		configs = append(configs, userCfg)
	}

	if cfg.systemSSHConfig != "" {
		// Not os.OpenInRoot: some distros (e.g. NixOS) make the system config a symlink
		// outside /etc/ssh, which it would refuse.
		systemFile, err := os.Open(cfg.systemSSHConfig)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, fmt.Errorf("open %s: %w", cfg.systemSSHConfig, err)
		default:
			defer systemFile.Close()
			systemCfg, err := sshconfig.Decode(systemFile, sshconfig.WithHome(home), sshconfig.WithIncludeDir(filepath.Dir(cfg.systemSSHConfig)))
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", cfg.systemSSHConfig, err)
			}
			configs = append(configs, systemCfg)
		}
	}

	return configs, nil
}

// rewriteFromSSHConfig rewrites the host, username, port and identity file from ~/.ssh/config
// and, if enabled, the system-wide ssh config. Tokens such as %h are expanded like OpenSSH
// does for each keyword. A username, port or identity file given on the command line is kept.
//
// It returns an error if a config cannot be parsed or a value has an unknown token.
func (cfg *Config) rewriteFromSSHConfig() error {
	configs, err := cfg.readSSHConfigs()
	if err != nil {
		return err
	}

	home := filepath.Dir(cfg.sshDir)
	alias := cfg.host
	vals := sshconfig.Lookup(alias, configs...)

	if v := vals.Get("HostName"); v != "" {
		host, err := sshconfig.ExpandTokens(v, map[byte]string{'h': alias})
		if err != nil {
			return fmt.Errorf("ssh config HostName for %s: %w", alias, err)
		}
		cfg.host = host
	}

	if v := vals.Get("Port"); !cfg.portSet && v != "" {
		cfg.port = v
	}

	if v := vals.Get("User"); !cfg.usernameSet && v != "" {
		username, err := sshconfig.ExpandTokens(v, map[byte]string{
			'h': cfg.host, 'n': alias, 'p': cfg.port, 'u': cfg.localUser,
		})
		if err != nil {
			return fmt.Errorf("ssh config User for %s: %w", alias, err)
		}
		cfg.username = username
	}

	if v := vals.Get("IdentityFile"); cfg.identityFile == "" && v != "" {
		identityFile, err := sshconfig.ExpandTokens(v, map[byte]string{
			'd': home, 'h': cfg.host, 'n': alias, 'p': cfg.port, 'r': cfg.username, 'u': cfg.localUser,
		})
		if err != nil {
			return fmt.Errorf("ssh config IdentityFile for %s: %w", alias, err)
		}
		cfg.identityFile = identityFile
	}

	return nil
}
