# WASH — Web Accessible Shell

> Cross-platform Go application providing remote shell access via WebSocket and REST API.

## Quick Links

- 🧪 **Auto-tests Guide**: [docs/AUTO_TESTS.md](docs/AUTO_TESTS.md)
- 📖 **Full English Docs**: [docs/README.md](docs/README.md)
- 🔧 **Configuration Guide**: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)
- 📡 **API Reference**: [docs/API_REFERENCE.md](docs/API_REFERENCE.md)
- 🧩 **Endpoint Summary**: [docs/ENDPOINTS_SUMMARY.md](docs/ENDPOINTS_SUMMARY.md)
- 🧪 **Testing Guide**: [docs/TESTING.md](docs/TESTING.md)
- 🛠 **Troubleshooting**: [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

## Overview

**WASH** is a cross-platform Go application that provides remote shell access through two interfaces:

1. **WebSocket** (`/ws`) — interactive, full-duplex shell session with a web-based terminal UI.
2. **REST API** (`POST /api/command`) — single-command execution via HTTP POST with JSON request/response.

## Quick Start

```bash
# Build
go build -o WASH

# Run with token authentication
./WASH -token=MY_SECRET_TOKEN -port=9091

# Run with OS authentication
./WASH -os-auth -port=9091
```

Open a browser and navigate to `http://localhost:9091/` to access the web terminal.

## Features

- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Interactive shell via WebSocket with real-time output
- ✅ REST API for scripted command execution
- ✅ Token-based authentication
- ✅ OS user/password authentication
- ✅ Graceful shutdown on SIGINT/SIGTERM
- ✅ Multiple concurrent sessions
- ✅ Automatic ping/keep-alive (30s interval)

## Documentation Structure

| File | Description |
|------|-------------|
| [docs/README.md](docs/README.md) | Full English documentation |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Configuration guide (YAML, .env, CLI) |
| [docs/API_REFERENCE.md](docs/API_REFERENCE.md) | Complete API reference |
| [docs/ENDPOINTS_SUMMARY.md](docs/ENDPOINTS_SUMMARY.md) | Endpoint quick reference |
| [docs/TESTING.md](docs/TESTING.md) | Test results and known issues |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Troubleshooting guide |
| [docs/AUTO_TESTS.md](docs/AUTO_TESTS.md) | Auto-tests guide (Russian) |

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
