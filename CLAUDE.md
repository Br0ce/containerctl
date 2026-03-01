# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
make setup       # Install golangci-lint and tidy dependencies (run once)
make build       # Build binary to ./bin/cctl
make run         # Run via go run ./cmd
make test        # Run tests (clears cache, parallel=4, short flag)
make test-v      # Run tests with verbose output and coverage
make test-race   # Run tests with race detector (used in CI)
make lint        # Run golangci-lint
make format      # Format with go fmt
make tidy        # Tidy and vendor dependencies
```

Run a single test:
```sh
go test ./pkg/client/... -run TestFunctionName -v
```

## Architecture

`cctl` is a k9s-inspired terminal UI for Docker container management, built on tview/tcell with a layered architecture:

```
cmd/             → CLI entry point (Cobra)
pkg/
  api/console/   → TUI layer (tview views, event handling, update loop)
  client/        → Docker API wrapper
  container/     → Data models (Container struct)
```

**Data flow:** `client.Shorts(ctx)` fetches container list → `PopulateContainersView()` renders table → user selects container → `client.Logs(ctx, id)` returns `iter.Seq2[string, error]` → `PopulateLogsView()` streams ANSI-translated logs.

**Key patterns:**
- The update loop in `api.go` refreshes every 3 seconds via a goroutine using `app.QueueUpdateDraw()` for thread-safe UI updates; it only runs when the containers page is active.
- Logs are modeled as `iter.Seq2[string, error]` (Go 1.22+ iterator pattern) and consumed with a range loop.
- Context-based cancellation (`context.WithCancel`) propagates through all async ops.

**Keyboard shortcuts:** `q` quit, `l` view selected container logs, `Esc` return to containers page.

**State colors:** green (running), red (exited), yellow (other states).

## Workflow

Before making any changes, provide a brief overview of the key idea behind your intended implementation and ask the user whether this approach matches their intent. Wait for confirmation before proceeding.

## Dependencies

Uses a vendored `vendor/` directory. After adding/removing dependencies, run `make tidy`.

The project targets Go 1.25+ and is tested on ubuntu-latest and macos-latest in CI.
