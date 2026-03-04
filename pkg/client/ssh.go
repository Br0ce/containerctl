package client

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/user"
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
func NewSSHClient(host, username, identityFile string) (*ssh.Client, error) {
	name, addr, err := resolveHost(host)
	if err != nil {
		return nil, fmt.Errorf("get host address: %w", err)
	}

	// If the username is provided, it will be used for authentication.
	if username != "" {
		if name != "" {
			return nil, fmt.Errorf("username provided in both host and argument")
		}
		name = username
	}

	curUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}

	if name != "" {
		// A username is provided: We will use password authentication.
		cfg, err := getUserConfig(name, curUser)
		if err != nil {
			return nil, fmt.Errorf("get SSH client config for username/password: %w", err)
		}
		return ssh.Dial("tcp", addr, cfg)
	}

	// No username is provided: We will use a private key for authentication.
	cfg, err := getPrivateKeyConfig(identityFile, curUser)
	if err != nil {
		return nil, fmt.Errorf("get SSH client config for private key: %w", err)
	}

	return ssh.Dial("tcp", addr, cfg)
}

// getPrivateKeyConfig returns an ssh.ClientConfig for the identity file if provided, or
// uses the default identity files in the ~/.ssh directory.
func getPrivateKeyConfig(identityFile string, curUser *user.User) (*ssh.ClientConfig, error) {
	sshDir := filepath.Join(curUser.HomeDir, ".ssh")

	file, err := openIdentityFile(identityFile, sshDir)
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

	callback, err := knownhosts.New(filepath.Join(sshDir, "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("get host key callback: %w", err)
	}

	return &ssh.ClientConfig{
		User: curUser.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: callback,
	}, nil

}

// openIdentityFile tries to open the identity file from the given filename
// or if filename is empty, the default ones in the ~/.ssh directory.
// It returns an error if the file cannot be opened or if the filename is outside of the ~/.ssh directory.
func openIdentityFile(filename, sshDir string) (*os.File, error) {
	if filename != "" {
		return os.OpenInRoot(sshDir, filepath.Base(filename))
	}

	// No filename provided, try the default ones.
	for _, f := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		idFile, err := os.OpenInRoot(sshDir, f)
		if err == nil {
			return idFile, nil
		}
	}

	return nil, fmt.Errorf("open default identity file")
}

// getUserConfig returns an ssh.ClientConfig for the given username, using a callback
// for password authentication.
func getUserConfig(username string, curUser *user.User) (*ssh.ClientConfig, error) {
	sshDir := filepath.Join(curUser.HomeDir, ".ssh")
	callback, err := knownhosts.New(filepath.Join(sshDir, "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("get host key callback: %w", err)
	}

	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (secret string, err error) {
				fmt.Print("Enter password: ")

				bytePassword, err := term.ReadPassword(syscall.Stdin)
				if err != nil {
					return "", fmt.Errorf("read password: %w", err)
				}

				return string(bytePassword), nil
			}),
		},
		HostKeyCallback: callback,
	}, nil
}

// resolveHost parses the host string and returns the username, host address, and any error.
// If the host string does not contain a port, it defaults to 22.
func resolveHost(host string) (string, string, error) {
	u, err := url.Parse("tcp://" + host)
	if err != nil {
		return "", "", fmt.Errorf("parse host URL: %w", err)
	}
	if u.Port() == "" {
		return u.User.Username(), net.JoinHostPort(u.Host, "22"), nil
	}
	return u.User.Username(), u.Host, nil
}
