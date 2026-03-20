package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// NewSSHClient creates a new SSH client from the given config.
//
// Settings from ~/.ssh/config are applied first, overriding any values in cfg.
// Host keys are verified against ~/.ssh/known_hosts.
//
// Authentication is attempted in order:
//  1. Password prompt, if cfg.AskPassword() is true.
//  2. SSH agent via SSH_AUTH_SOCK, if the socket is available and has signers.
//  3. Identity file (defaults: id_ed25519, id_rsa, id_ecdsa); prompts for passphrase if needed.
func NewSSHClient(cfg *Config) (*ssh.Client, error) {
	callback, err := knownhosts.New(filepath.Join(cfg.SSHDir(), "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("get host key callback: %w", err)
	}

	// Apply ~/.ssh/config overrides before any auth decisions.
	cfgFile, err := os.OpenInRoot(cfg.sshDir, "config")
	if err == nil {
		defer cfgFile.Close()
		if err := cfg.rewriteFromSSHConfig(cfgFile); err != nil {
			return nil, fmt.Errorf("rewrite from ssh config: %w", err)
		}
	}

	baseCfg := ssh.ClientConfig{
		HostKeyCallback: callback,
		User:            cfg.Username(),
	}

	if cfg.AskPassword() {
		return cfg.sshClientFromPassword(baseCfg)
	}

	// Try agent auth if SSH_AUTH_SOCK is available.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		client, err := cfg.sshClientFromAgent(sock, baseCfg)
		if err == nil {
			return client, nil
		}
	}

	// Fall back to key file auth.
	return cfg.sshClientFromKeyFile(baseCfg)
}

// sshClientFromPassword creates an SSH client using password authentication.
func (cfg *Config) sshClientFromPassword(cc ssh.ClientConfig) (*ssh.Client, error) {
	callback := func() (secret string, err error) {
		fmt.Print("Enter password: ")

		bytePassword, err := term.ReadPassword(syscall.Stdin)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}

		return string(bytePassword), nil
	}
	cc.Auth = []ssh.AuthMethod{ssh.PasswordCallback(callback)}
	return ssh.Dial("tcp", cfg.Addr(), &cc)
}

// sshClientFromAgent creates an SSH client using agent authentication with the given socket and client config.
func (cfg *Config) sshClientFromAgent(sock string, cc ssh.ClientConfig) (*ssh.Client, error) {
	//nolint:gosec // G704: SSH_AUTH_SOCK is a well-known env var pointing to a Unix socket, not a network resource
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("dial ssh agent sock: %w", err)
	}
	defer conn.Close()

	signers, err := agent.NewClient(conn).Signers()
	if err != nil || len(signers) == 0 {
		return nil, fmt.Errorf("no signers from agent")
	}

	cc.Auth = []ssh.AuthMethod{ssh.PublicKeys(signers...)}
	client, err := ssh.Dial("tcp", cfg.Addr(), &cc)
	if err != nil {
		return nil, fmt.Errorf("dial ssh: %w", err)
	}

	return client, nil
}

// sshClientFromKeyFile creates an SSH client using key file authentication with the given client config.
func (cfg *Config) sshClientFromKeyFile(cc ssh.ClientConfig) (*ssh.Client, error) {
	file, err := cfg.openIdentityFile()
	if err != nil {
		return nil, fmt.Errorf("open identity file: %w", err)
	}
	defer file.Close()

	key, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read identity file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if !errors.As(err, new(*ssh.PassphraseMissingError)) {
			return nil, fmt.Errorf("parse private key: %w", err)
		}

		fmt.Printf("Enter passphrase for key %s: ", file.Name())
		passphrase, err := term.ReadPassword(syscall.Stdin)
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("read passphrase: %w", err)
		}

		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
		if err != nil {
			return nil, fmt.Errorf("parse private key with passphrase: %w", err)
		}
	}

	cc.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	return ssh.Dial("tcp", cfg.Addr(), &cc)
}
