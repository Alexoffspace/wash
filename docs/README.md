# WASH — Web Accessible Shell

> Cross-platform Go application providing remote shell access via WebSocket and REST API.

## Configuration File

The application supports configuration via `config.yaml` and `.env` files.

### Configuration Priority

1. **CLI arguments** (highest priority)
2. **config.yaml**
3. **Environment variables**
4. **Default values**

### config.yaml

Create a `config.yaml` file in the application directory:

```yaml
# Enable OS authentication (true/false)
os_auth: true

# Environment variable name for the token
token: WASH_TOKEN

# Port on which the application will run
port: 9091

# Listen on 0.0.0.0 (true) or 127.0.0.1 (false)
allow_0: false
```

### .env

Create a `.env` file to store sensitive data (tokens, passwords):

```bash
# Environment variables for WASH
WASH_TOKEN=your_token_here
```

The application automatically loads `.env` and `config.yaml` if they exist.

### Examples

**Using config.yaml (no CLI arguments):**
```bash
./WASH
```

**CLI overrides config.yaml:**
```bash
./WASH -token=cli_token -port=8080
```

**Allow 0.0.0.0 via CLI:**
```bash
./WASH -allow-0
```

## Overview

**WASH** is a cross-platform Go application that provides remote shell access through two interfaces:

1. **WebSocket** (`/ws`) — interactive, full-duplex PTY-based shell session with a web-based xterm.js terminal UI.
2. **REST API** (`POST /api/command`) — single-command execution via HTTP POST with JSON request/response.

The application runs as a standalone HTTP server and supports both token-based and OS user/password authentication.

## Features

- Cross-platform support (Linux, macOS, Windows)
- Interactive shell via WebSocket with PTY-based real-time output (xterm.js)
- Configurable shell (sh, bash, zsh, fish, etc.) via config.yaml
- REST API for scripted command execution
- Token-based authentication
- OS user/password authentication (Linux `su`, Windows PowerShell)
- Graceful shutdown on SIGINT/SIGTERM
- Multiple concurrent sessions
- Automatic ping/keep-alive (30s interval)
- Background images for light/dark themes
- Embedded static files (no external dependencies for the web UI)

## Installation

**Prerequisites:** Go 1.22 or later.

```bash
git clone <repository-url>
cd WAShell
go build -o WASH
```

The compiled binary `WASH` is ready to run.

## Quick Start

Run with token authentication:

```bash
./WASH -token=MY_SECRET_TOKEN -port=9091
```

Run with OS authentication enabled:

```bash
./WASH -os-auth -port=9091
```

Run with both:

```bash
./WASH -token=TOKEN1,TOKEN2 -os-auth -port=9091
```

Run listening on all interfaces (0.0.0.0):

```bash
./WASH -token=MY_SECRET_TOKEN -port=9091 -allow-0
```

Open a browser and navigate to `http://localhost:9091/` to access the web terminal.

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-token` | `""` | Auth token for API access. Comma-separated for multiple tokens. |
| `-port` | `8080` | Server HTTP port. |
| `-os-auth` | `false` | Enable OS user/password authentication. |
| `-allow-0` | `false` | Allow listening on 0.0.0.0 (default: 127.0.0.1). |
| `-max-msg-size` | `1048576` (1 MB) | Maximum WebSocket message size in bytes. |

**Note:** CLI arguments override values from `config.yaml`. See [Configuration File](#configuration-file) section for details.

## Authentication

### Token Authentication

Set one or more tokens via the `-token` flag:

```bash
./WASH -token=abc123,def456
```

**REST API** — pass the token in the `X-Auth-Token` header:

```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: abc123" \
  -d '{"command": "whoami"}'
```

**WebSocket** — send an authentication message as the first message after connecting:

```json
{
  "type": "auth",
  "login": "",
  "password": "abc123"
}
```

### OS User/Password Authentication

Enable with the `-os-auth` flag. Users authenticate with their OS username and password:

**REST API** — use HTTP Basic Auth:

```bash
curl -X POST http://localhost:9091/api/command \
  -u "username:password" \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

**WebSocket** — send login and password in the auth message:

```json
{
  "type": "auth",
  "login": "username",
  "password": "password"
}
```

> **Note:** On Windows, password verification is limited (only user existence is checked). On Linux, authentication is verified via `su`.

## API Reference

**See [API_REFERENCE.md](API_REFERENCE.md) for complete API documentation.**

Quick reference:

### REST API

- `GET /api/status` — System information (requires auth)
- `POST /api/command` — Execute shell command (requires auth)

### WebSocket API

- `ws://localhost:9091/ws` — Interactive shell session
- Message types: `auth`, `command`, `key`, `resize`, `ping`, `pong`
- Server responses: `auth_success`, `auth_error`, `output`, `system`, `error`, `pong`

### Testing

**See [TESTING.md](TESTING.md) for comprehensive test results and known issues.**

## Current status:
- ✅ REST API: All endpoints working
- ✅ WebSocket: Authentication + PTY-based output streaming working
- ✅ Rate limiting: Enforced (10 failed attempts per minute per IP)
- ✅ Configurable shell: bash, zsh, fish via config.yaml
- ✅ xterm.js terminal: full terminal emulation with ANSI support

## Project Structure

```
WASH/
├── main.go              # Entry point, HTTP server setup
├── config.yaml          # Configuration file (YAML)
├── .env                 # Environment variables (tokens, sensitive data)
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── integration_test.go  # Integration tests
├── pkg/
│   ├── api/
│   │   └── api.go       # REST API handler (/api/command, /api/status)
│   ├── auth/
│   │   └── auth.go      # Authentication logic (token + OS)
│   ├── shell/
│   │   └── shell.go     # PTY-based shell session, command execution
│   └── ws/
│       └── ws.go        # WebSocket session manager, PTY I/O, rate limiting
├── static/              # Embedded static files (CSS, JS, xterm.js, fonts, backgrounds)
└── templates/
    └── index.html       # Web terminal UI (xterm.js)
```

## Building from Source

```bash
# Build for current platform
go build -o WASH

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o WASH-linux
GOOS=windows GOARCH=amd64 go build -o WASH.exe
GOOS=darwin GOARCH=arm64 go build -o WASH-darwin

# Run tests
go test ./...
```

## Security Considerations

1. **Use HTTPS in production.** The application does not include TLS — use a reverse proxy (nginx, Caddy) in front.
2. **Configure CORS.** The WebSocket upgrader validates origins: in localhost mode only loopback origins are accepted; in `-allow-0` mode the request host must match the origin. For production behind a reverse proxy, ensure `X-Forwarded-Host` is set correctly.
3. **Strong tokens.** Use long, random tokens. Never use weak or guessable values.
4. **Protect .env file.** Ensure `.env` is not committed to version control and has proper file permissions.
5. **OS auth caution.** OS user/password authentication via `su` on Linux has limitations (reads password from stdin). On Windows, password verification is limited.
6. **Input validation.** Shell commands are executed directly — sanitize all inputs from untrusted sources.
7. **Session limits.** There is no built-in limit on concurrent sessions. Monitor resource usage.
