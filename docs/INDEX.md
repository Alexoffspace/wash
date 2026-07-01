# WASH Documentation Index

## Overview

WASH is a cross-platform Go application providing remote shell access via WebSocket and REST API.

**Quick Links**:
- 🚀 [Quick Start](README.md#quick-start)
- 📚 [API Reference](API_REFERENCE.md)
- 🧪 [Testing Results](TESTING.md)
- 🔧 [Configuration](CONFIGURATION.md)
- ⚙️ [Troubleshooting](TROUBLESHOOTING.md)

---

## Documentation Files

### [README.md](README.md)
Main documentation covering:
- Installation and setup
- Quick start guide
- Command-line options
- Authentication methods
- Project structure
- Security considerations
- Configuration file reference

**Start here** if you're new to WASH.

---

### [API_REFERENCE.md](API_REFERENCE.md)
Complete API documentation including:
- **REST API**
  - `GET /api/status` - System information
  - `POST /api/command` - Execute commands
- **WebSocket API**
  - `/ws` - Interactive shell sessions (PTY-based)
  - Message types and formats
  - Authentication flow
  - Error handling
- Authentication methods (token & OS auth)
- Rate limiting
- Security considerations
- Python example code

**Use this** when developing against WASH.

---

### [TESTING.md](TESTING.md)
Comprehensive test results and known issues:
- Test summary table
- Detailed test results for each endpoint
- Known issues and their severity
- Root cause analysis
- Reproduction steps
- Recommended fixes
- Manual testing procedures

**Check this** to understand what's working and what needs fixing.

---

### [ENDPOINTS_SUMMARY.md](ENDPOINTS_SUMMARY.md)
Quick reference for all endpoints:
- Endpoint table with status
- Detailed endpoint documentation
- Request/response examples
- Authentication methods
- Status summary (working/issues)
- Configuration quick reference
- Limits and constraints

**Use this** for quick endpoint lookup.

---

### [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
Solutions for common problems:
- Server won't start
- Authentication failures
- CORS errors
- Custom shell configuration
- WebSocket ping/pong issues
- Debugging tips
- Performance tuning

**Consult this** when something isn't working.

---

### [CONFIGURATION.md](CONFIGURATION.md)
Configuration guide:
- Config file formats (YAML, .env)
- Configuration priority
- Command-line flags
- Environment variables
- Examples

**Read this** to configure WASH.

---

### [AUTO_TESTS.md](AUTO_TESTS.md)
Automated test suite:
- Test framework setup
- Test cases
- Running tests
- Test coverage

**Use this** for automated testing.

---

## Quick Start

### Installation
```bash
git clone <repository>
cd WAShell
go build -o WASH
```

### Run Server
```bash
# Token authentication
./WASH -token=YOUR_TOKEN -port=9091

# OS authentication
./WASH -os-auth -port=9091
```

Open http://localhost:9091 in your browser, enter the token (or OS credentials), and click Connect.

### Test REST API
```bash
# Get system status
curl -H "X-Auth-Token: YOUR_TOKEN" http://localhost:9091/api/status

# Execute command
curl -X POST http://localhost:9091/api/command \
  -H "X-Auth-Token: YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

### Test WebSocket
```python
import asyncio, websockets, json

async def test():
    async with websockets.connect('ws://localhost:9091/ws') as ws:
        await ws.send(json.dumps({'type': 'auth', 'password': 'YOUR_TOKEN'}))
        # auth_success
        print(await ws.recv())
        # system message (shell started)
        print(await ws.recv())
        # send a command
        await ws.send(json.dumps({'type': 'key', 'content': 'whoami\n'}))
        # output (raw PTY data with ANSI codes)
        print(await ws.recv())

asyncio.run(test())
```

---

## Current Status

### ✅ Working
- REST API endpoints (`/api/status`, `/api/command`)
- Token-based authentication
- OS authentication (Basic Auth)
- WebSocket connection and authentication
- PTY-based command output streaming
- System information retrieval
- Command execution (REST API)
- Rate limiting (10 failed attempts per minute per IP)
- Auto-reconnect with exponential backoff
- Native WebSocket keepalive (PingMessage)

### ⚠️ Issues
- (none known)

### 📋 See Also
- [TESTING.md](TESTING.md) - Detailed test results
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Known issues and fixes

---

## Architecture

```
WAShell/
├── main.go              # Server setup
├── pkg/
│   ├── api/             # REST API handlers
│   ├── auth/            # Authentication logic
│   ├── shell/           # PTY-based shell sessions
│   └── ws/              # WebSocket + PTY I/O
├── static/              # Embedded static files (xterm.js, CSS, fonts)
└── templates/
    └── index.html       # Web UI (xterm.js terminal)
```

---

## Key Features

- 🔐 **Dual Authentication**: Token-based + OS user/password
- 🌐 **Two Interfaces**: REST API + WebSocket
- 🔄 **Interactive Sessions**: Full PTY-based shell via WebSocket (xterm.js)
- 📊 **System Info**: CPU, memory, disk, uptime
- 🛡️ **CORS Protection**: Origin validation
- 🔌 **Keep-Alive**: Native WebSocket PingMessage every 30s
- 📝 **Logging**: Detailed debug logs
- ⚙️ **Configurable shell**: sh, bash, zsh, fish via config.yaml
- 🖼️ **Background images**: Random backgrounds for light/dark themes

---

## Authentication

### Token Auth
```bash
# REST API
curl -H "X-Auth-Token: YOUR_TOKEN" http://localhost:9091/api/status

# WebSocket
{"type": "auth", "password": "YOUR_TOKEN"}
```

### OS Auth
```bash
# REST API
curl -u "username:password" http://localhost:9091/api/command

# WebSocket
{"type": "auth", "login": "username", "password": "password"}
```

---

## Support

- 📖 Read the [README.md](README.md) for general information
- 🔍 Check [TESTING.md](TESTING.md) for what's working/broken
- 🛠️ See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for solutions
- 📚 Consult [API_REFERENCE.md](API_REFERENCE.md) for API details

---

## Document Status

| Document | Status | Last Updated |
|----------|--------|--------------|
| README.md | Updated | 2026-06-30 |
| API_REFERENCE.md | Updated | 2026-06-30 |
| TESTING.md | Updated | 2026-06-30 |
| ENDPOINTS_SUMMARY.md | Updated | 2026-06-30 |
| TROUBLESHOOTING.md | Updated | 2026-06-30 |
| CONFIGURATION.md | Updated | 2026-06-30 |
| AUTO_TESTS.md | Updated | 2026-06-30 |

---

## Version

- **WASH**: 2.0.0
- **API**: 1.0
- **Documentation**: 2026-06-30
