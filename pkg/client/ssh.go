package client

import (
	"fmt"
	"io"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// NewSSHClient creates a new SSH client for the given host, username, and identity file.
//
// The host can be in the format of "example.com", "user@example.com:22", or any valid
// combination of username, hostname, and port. If the port is not specified, it defaults to 22.
//
// If a username is provided, the client will use password authentication. If the username is
// provided in both the host string and the argument, it will return an error.
//
// If no username is provided, it will try to use an identity file for authentication.
//
// If the identity file is provided, it will be used. Any identity file outside of ~/.ssh will
// not be accepted for security reasons.
// Otherwise, the default identity files in the .ssh directory will be tried,
// e.g., id_ed25519, id_rsa, and id_ecdsa.
//
// In either case, the client will check the host key against the known_hosts file in the ~/.ssh directory.
func NewSSHClient(cfg Config) (*ssh.Client, error) {
	callback, err := knownhosts.New(filepath.Join(cfg.SSHDir(), "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("get host key callback: %w", err)
	}

	authMethod, err := getAuthMethod(cfg)
	if err != nil {
		return nil, fmt.Errorf("get auth method: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		HostKeyCallback: callback,
		User:            cfg.Username(),
		Auth:            authMethod,
	}

	return ssh.Dial("tcp", cfg.host, sshCfg)
}

func getAuthMethod(cfg Config) ([]ssh.AuthMethod, error) {
	if cfg.AskPassword() {
		callback := func() (secret string, err error) {
			fmt.Print("Enter password: ")

			bytePassword, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return "", fmt.Errorf("read password: %w", err)
			}

			return string(bytePassword), nil
		}
		return []ssh.AuthMethod{
			ssh.PasswordCallback(callback),
		}, nil
	}

	// Try to open the identity file.
	file, err := cfg.OpenIdentityFile()
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
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return []ssh.AuthMethod{
		ssh.PublicKeys(signer),
	}, nil
}
