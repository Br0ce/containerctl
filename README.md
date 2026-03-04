# containerctl

A terminal UI for Docker container management.

## Overview

`containerctl` removes the friction from managing containers over SSH.
It’s a lightweight CLI that lets you instantly inspect, control, and tail logs from containers — perfect for remote servers where you want speed, not tooling overhead.

It sits as a thin layer between the [tview](https://github.com/rivo/tview) TUI framework and the [Moby](https://github.com/moby/moby) Docker client. It supports connecting to a remote Docker daemon directly over SSH, so you can manage containers on a remote host without SSHing in manually.

Heavily inspired by [k9s](https://github.com/derailed/k9s)

---

![Build Status](https://github.com/Br0ce/containerctl/actions/workflows/ci.yml/badge.svg)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/Br0ce/containerctl)](https://github.com/Br0ce/containerctl)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/containerctl.svg)](https://pkg.go.dev/github.com/Br0ce/containerctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/Br0ce/containerctl)](https://goreportcard.com/report/github.com/Br0ce/containerctl)
![GitHub License](https://img.shields.io/github/license/Br0ce/containerctl)

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
