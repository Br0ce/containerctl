# containerctl

A lightweight terminal UI for Docker container management, built for speed over remote SSH connections

## Overview

`containerctl` removes the friction from managing containers over SSH.
It’s a lightweight CLI that lets you instantly inspect, control, and tail logs from containers — perfect for remote servers where you want speed, not tooling overhead.

It sits as a thin layer between the [tview](https://github.com/rivo/tview) TUI framework and the [Moby](https://github.com/moby/moby) Docker client. It supports connecting to a remote Docker Engine API directly over SSH, so you can manage containers on a remote host without SSHing in manually.

## Behavior

#### Local mode (no `--host`)

Connects to a local Docker API-compatible socket using the first available option:

1. `DOCKER_HOST`, `DOCKER_API_VERSION`, `DOCKER_CERT_PATH` or `DOCKER_TLS_VERIFY` environment variable
2. `--docker-host` flag
3. Default: `unix:///var/run/docker.sock`

#### Remote mode (`--host`)

Connects securely over SSH. Host key verification is enforced via `~/.ssh/known_hosts`.

If a `~/.ssh/config` entry matches the given host, values for `HostName`, `Port`, `User`, or `IdentityFile` are read from it and take precedence over command-line arguments to replicate OpenSSH behavior.

**Username** is resolved in this order:
1. `--username` flag
2. `user@hostname` syntax in `--host`
3. Current system user

**Authentication** uses password if `--ask-password` is set, otherwise a private key resolved in this order:
1. `--identity-file` flag (must be inside `~/.ssh/`)
2. `~/.ssh/config` `IdentityFile` entry for the host
3. Default keys: `~/.ssh/id_ed25519`, `id_rsa`, `id_ecdsa`

**Docker Host** on the remote host (note: `DOCKER_HOST`, `DOCKER_API_VERSION`, `DOCKER_CERT_PATH` and `DOCKER_TLS_VERIFY` are ignored in remote mode):
1. `--docker-host` flag
2. Default: `unix:///var/run/docker.sock`

## Examples

```bash
  # Connect to local Docker
  containerctl

  # Connect to remote host using default SSH key from ~/.ssh/config
  containerctl --host my-host

  # Connect to remote host using with SSH key
  containerctl --host my-host:23 --identity-file ~/.ssh/id_rsa

  # Connect to remote host with username embedded in host and password prompted
  containerctl --host username@my-host --ask-password true
```

---

![Build Status](https://github.com/Br0ce/containerctl/actions/workflows/ci.yml/badge.svg)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/Br0ce/containerctl)](https://github.com/Br0ce/containerctl)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/containerctl.svg)](https://pkg.go.dev/github.com/Br0ce/containerctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/Br0ce/containerctl)](https://goreportcard.com/report/github.com/Br0ce/containerctl)

---

## Requirements

- Go 1.26+

## Installation

```sh
git clone https://github.com/Br0ce/containerctl.git
cd containerctl
make build
```

The binary will be placed at `./bin/containerctl`.

## Usage

```sh
./bin/containerctl
# or
make run
```

## Development

Install dependencies and tooling:

```sh
make setup
```

Available make targets:

| Target         | Description                          |
|----------------|--------------------------------------|
| `build`        | Build the binary to `./bin/`         |
| `run`          | Run directly via `go run`            |
| `test`         | Run tests (short, parallel)          |
| `test-v`       | Run tests with verbose output        |
| `test-race`    | Run tests with race detector         |
| `lint`         | Run golangci-lint                    |
| `format`       | Format code with `go fmt`            |
| `tidy`         | Tidy and vendor dependencies         |
| `clean`        | Remove build artifacts               |

## Acknowledgements

- Inspired by [k9s](https://github.com/derailed/k9s)
- TUI built with [tview](https://github.com/rivo/tview) by [@rivo](https://github.com/rivo)
- Docker integration via the [Moby](https://github.com/moby/moby) Go client

## License

Apache 2.0 — see [LICENSE](LICENSE).
