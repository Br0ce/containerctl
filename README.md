# containerctl

A terminal UI for managing containers.

## Overview

`containerctl` provides a lightweight, interactive terminal interface for working with containers.

---

![Build Status](https://github.com/Br0ce/containerctl/actions/workflows/ci.yml/badge.svg)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/Br0ce/containerctl)](https://github.com/Br0ce/containerctl)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/containerctl.svg)](https://pkg.go.dev/github.com/Br0ce/containerctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/Br0ce/containerctl)](https://goreportcard.com/report/github.com/Br0ce/containerctl)
![License](https://img.shields.io/badge/license-MIT-green)

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

## License

MIT — see [LICENSE](LICENSE).
