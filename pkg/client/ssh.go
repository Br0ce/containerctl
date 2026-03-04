package client

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func NewSSHClient(hostname string) (*ssh.Client, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	// #nosec G304 -- filename is controlled and not user input
	key, err := os.ReadFile(filepath.Join(sshDir, "id_rsa"))
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	callback, err := knownhosts.New(filepath.Join(sshDir, "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("get known hosts: %w", err)
	}

	config := &ssh.ClientConfig{
		User: currentUser.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: callback,
	}

	return ssh.Dial("tcp", hostname+":22", config)
}
