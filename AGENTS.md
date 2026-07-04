# WASH (Web Accessible Shell) — Agent Guide

Single-module Go 1.25 app (`module WASH`). Entrypoint: `main.go`. Embedded web UI via `//go:embed templates/* static/*`.

## Build & Run

```
go build -o WASH
./WASH -token=SECRET -port=9091         # token auth
./WASH -os-auth -port=9091               # OS auth (su/PowerShell)
./WASH -token=TOKEN1,TOKEN2 -os-auth -allow-0   # both + all interfaces
```

Cross-compile: `GOOS=linux GOARCH=amd64 go build -o WASH-linux` (also `windows`, `darwin`).

Config priority: CLI flags > `config.yaml` > `.env` > defaults.

## Test

```
go test -v ./...                        # all tests
go test -v ./pkg/ws/                    # ws pkg only (CORS tests)
go test -v -run TestShellSession .      # single test
go test -v -race ./...                  # with race detector
```

- All integration tests in one file: `integration_test.go` (package `main`).
- Unit tests: `pkg/api/api_test.go` (package `api`).
- **Known flaky:** `TestShellSession`, `TestRunCommand/pwd` — output depends on work dir.
- **WebSocket tests are placeholders** — they verify server starts, not real WS connections.

## Lint & Format (no CI configured)

```
gofmt -l .     # check formatting (no .golangci.yml exists)
go vet ./...
```

## Known Issues

- **WebSocket output streaming** — 10ms `select` timeout in `ReadStdout` may miss early output (`pkg/shell/shell.go:ReadStdout`). Increase timeout or use blocking read.
- **Rate limiting** — Duplicate rate-limiter in `api.go` (`APIAuthAttemptTracker`) separate from `ws.go`. Both work correctly now.
- **Windows ConPTY** — `CreatePseudoConsole` requires Windows 10 1809+ / Windows Server 2019+. Falls back to pipe-based session on older Windows.

## Architecture

| Package | File | Role |
|---------|------|------|
| `pkg/config` | `config.go` | YAML + .env loader (BOM-safe) |
| `pkg/auth` | `auth.go` | Token + OS auth (su/PowerShell) |
| `pkg/shell` | `shell.go` | `RunCommand` (one-shot) + `Session` interface |
| `pkg/shell` | `session_unix.go` | Unix PTY session (creack/pty) |
| `pkg/shell` | `session_windows.go` | Windows session (ConPTY + pipe fallback, OEM codepage decoding) |
| `pkg/api` | `api.go` | REST: `POST /api/command`, `GET /api/status` (cross-platform metrics) |
| `pkg/ws` | `ws.go` | WebSocket: `GET /ws`, session lifecycle, rate limiting, PTY I/O |

Routes (stdlib `http.ServeMux`): `GET /` → embedded UI, `/ws` → WS, `/api/command` → REST, `/api/status` → status, `/static/` → embedded static.

## Conventions

- **Never build (`go build`) or start the server** — the user does this themselves.
- **Conventional Commits:** `feat(scope):`, `fix(scope):`, `docs:`, etc.
- **100% test coverage** required for new features.
- Table-driven tests preferred.
- UI: semantic HTML5, BEM CSS, ES6+ JS, xterm.js terminal.
- Comments must be in English (some source has Russian comments — do not add more).
- `docs/` dir for docs (Markdown), `config.yaml` for config.
- `.nnb/` is nanobot AI agent workspace — not application code.

## Dependencies

- `github.com/gorilla/websocket v1.5.3` (direct)
- `github.com/creack/pty v1.1.24` (direct, Unix PTY)
- `golang.org/x/sys v0.46.0` (direct, Windows ConPTY)
- `golang.org/x/text v0.38.0` (direct, OEM codepage decoding)
- `gopkg.in/yaml.v3 v3.0.1` (indirect)
