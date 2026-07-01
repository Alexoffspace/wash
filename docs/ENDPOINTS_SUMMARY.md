# WASH Endpoints Summary

## Quick Reference

### REST API Endpoints

| Method | Endpoint | Auth | Status | Purpose |
|--------|----------|------|--------|---------|
| GET | `/api/status` | Required | ✅ Working | Get system information |
| POST | `/api/command` | Required | ✅ Working | Execute shell command |

### WebSocket Endpoint

| Protocol | Endpoint | Auth | Status | Purpose |
|----------|----------|------|--------|---------|
| WS | `/ws` | Required | ✅ Working | Interactive PTY-based shell session (xterm.js) |

---

## Endpoint Details

### GET `/api/status`

**Description**: Returns system information and server status

**Authentication**: Required (token or OS auth)

**Request**:
```bash
curl -H "X-Auth-Token: 123" http://localhost:9091/api/status
```

**Response** (200 OK):
```json
{
  "hostname": "fishnet",
  "ip_addresses": ["192.168.1.100/24"],
  "os": "linux",
  "os_version": "6.8.0-124-generic",
  "architecture": "amd64",
  "time": "2026-06-24 15:47:12 +04",
  "uptime": "15:47:12 up 8 days, 7:30",
  "wash_status": {
    "status": "running",
    "version": "0.1.0",
    "auth_tokens": 1
  },
  "cpu": {
    "load1": "2.13",
    "load5": "1.72",
    "load15": "1.50",
    "cores": 8,
    "usage_pct": 23.5
  },
  "memory": {
    "total": 32982347776,
    "used": 9819123712,
    "free": 3100688384,
    "used_pct": 29.77,
    "swap_total": 23163224064,
    "swap_used": 0
  },
  "disk": {
    "total": 234620301312,
    "used": 155016876032,
    "free": 79603425280,
    "used_pct": 66.07,
    "mount_point": "/"
  },
  "processes": 359,
  "user": "unknown"
}
```

**Error Responses**:
- `401` - Missing or invalid authentication
- `500` - Server error

---

### POST `/api/command`

**Description**: Execute a shell command and return output

**Authentication**: Required (token or OS auth)

**Request Body**:
```json
{
  "command": "ls -la /tmp"
}
```

**Request Example**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": "whoami"}'
```

**Response** (200 OK):
```json
{
  "stdout": "belov\n",
  "stderr": "",
  "exit_code": 0
}
```

**Error Responses**:
- `400` - Invalid request (empty command, invalid JSON)
- `401` - Missing or invalid authentication
- `429` - Rate limited (too many failed auth attempts)
- `500` - Server error

**Validation**:
- Command cannot be empty
- Command is required in request body

**Rate Limiting**:
- 10 failed authentication attempts trigger rate limiting
- Rate limit window: 1 minute (attempts counter resets after 60 seconds)

---

### WS `/ws`

**Description**: Interactive shell session via WebSocket

**Authentication**: Required (token or OS auth)

**Connection**:
```javascript
const ws = new WebSocket('ws://localhost:9091/ws');
```

**Auth Message** (must be sent within 5 seconds):
```json
{
  "type": "auth",
  "password": "123"
}
```

**Auth Response** (Success):
```json
{
  "type": "auth_success",
  "session": "sess-1782301685089012424",
  "user": "token-user",
  "hostname": "fishnet",
  "ip": "192.168.1.100",
  "timestamp": "15:47"
}
```

**Auth Response** (Error):
```json
{
  "type": "auth_error",
  "content": "Authentication failed: empty credentials",
  "timestamp": "15:47"
}
```

**Send Command**:
```json
{
  "type": "command",
  "command": "whoami"
}
```

**Receive Output** (raw PTY data with ANSI):
```json
{
  "type": "output",
  "content": "belov\n",
  "timestamp": "15:47"
}
```

**Send Keystroke** (PTY, recommended):
```json
{
  "type": "key",
  "content": "whoami\n"
}
```

**Resize Terminal**:
```json
{
  "type": "resize",
  "cols": 120,
  "rows": 40
}
```

**Other Messages**:
- `system` - System notifications
- `error` - Error messages
- `ping` - Legacy keepalive (server now uses native WebSocket PingMessage)

**Client Features**:
- Auto-reconnect with exponential backoff (1s–30s max)
- Terminal resize reporting via `resize` message
- System status polling every 5s

---

## Authentication Methods

### Token Authentication

**REST API**:
```bash
curl -H "X-Auth-Token: YOUR_TOKEN" http://localhost:9091/api/status
```

**WebSocket**:
```json
{
  "type": "auth",
  "password": "YOUR_TOKEN"
}
```

### OS Authentication (Basic Auth)

**REST API**:
```bash
curl -u "username:password" http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

**WebSocket**:
```json
{
  "type": "auth",
  "login": "username",
  "password": "password"
}
```

---

## Status Summary

### ✅ Working

- `GET /api/status` - Returns system information
- `POST /api/command` - Executes commands and returns output
- WebSocket session - Connects, authenticates, and streams PTY output
- Token-based auth - Works for both REST and WebSocket
- OS authentication - Works for both REST and WebSocket

### ⚠️ Partial/Issues

(none)

---

## Configuration

### Start Server

```bash
# Token auth only
./WASH -port=9091

# OS auth only
./WASH -os-auth -port=9091

# Both auth methods
./WASH -os-auth -port=9091

# Custom port
./WASH -os-auth -port=9092

# Listen on all interfaces
./WASH -os-auth -port=9091 -allow-0
```

### Environment Variables

- `WASH_TOKEN` - Authentication token (from `.env` or environment)
- No `PORT` env var — use `-port` flag or `config.yaml`

---

## Limits

| Aspect | Limit |
|--------|-------|
| Command size | Unlimited |
| Output buffer | 4096 bytes per PTY read (channel buffer: 256 messages) |
| WebSocket buffer | 4096 bytes (read/write) |
| Send channel | 100 messages |
| Auth timeout | 5 seconds |
| Read deadline | 60 seconds |
| Write deadline | 30 seconds |
| Native ping | 30s (WebSocket PingMessage) |
| Rate limit threshold | 10 failures |
| Rate limit window | 1 minute |

---

## See Also

- [API_REFERENCE.md](API_REFERENCE.md) - Complete API documentation
- [TESTING.md](TESTING.md) - Test results and known issues
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Troubleshooting guide
