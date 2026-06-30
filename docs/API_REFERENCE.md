# WASH API Reference

## Overview

WASH provides two main interfaces for executing shell commands:
1. **REST API** (`/api/*`) - Synchronous command execution
2. **WebSocket** (`/ws`) - Interactive shell sessions

Both interfaces support multiple authentication methods:
- **Token-based**: Via `X-Auth-Token` header
- **OS Authentication**: Via HTTP Basic Auth (username and password)

---

## REST API

### Base URL
```
http://localhost:9091/api
```

### Authentication

#### Token Authentication
```bash
curl -H "X-Auth-Token: YOUR_TOKEN" http://localhost:9091/api/status
```

#### OS Authentication (Basic Auth)
```bash
curl -u "username:password" http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

---

### Endpoints

#### 1. GET `/api/status`

Returns system information and server status.

**Authentication**: Required (token or OS auth)

**Example Request**:
```bash
curl -H "X-Auth-Token: 123" http://localhost:9091/api/status
```

**Example Response**:
```json
{
  "hostname": "fishnet",
  "ip_addresses": [
    "192.168.1.100/24",
    "10.0.0.1/32"
  ],
  "os": "linux",
  "os_version": "6.8.0-124-generic",
  "architecture": "amd64",
  "time": "2026-06-24 15:47:12 +04",
  "uptime": "15:47:12 up 8 days,  7:30,  2 users,  load average: 2.13, 1.72, 1.50",
  "wash_status": {
    "status": "running",
    "start_time": "2026-06-24T14:47:12.686018279+04:00",
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

**Status Codes**:
- `200` - Success
- `401` - Missing or invalid authentication

---

#### 2. POST `/api/command`

Executes a shell command synchronously and returns the output.

**Authentication**: Required (token or OS auth)

**Request Body**:
```json
{
  "command": "ls -la /tmp"
}
```

**Example Request**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": "whoami"}'
```

**Example Response** (Success):
```json
{
  "stdout": "belov\n",
  "stderr": "",
  "exit_code": 0
}
```

**Example Response** (Error):
```json
{
  "error": "command is required"
}
```

**Validation**:
- Command cannot be empty
- Maximum command length: not specified (implementation dependent)

**Status Codes**:
- `200` - Command executed (check `exit_code` for success/failure)
- `400` - Invalid request (e.g., empty command)
- `401` - Missing or invalid authentication
- `429` - Rate limited (too many failed auth attempts)

**Rate Limiting**:
- **Per IP**: 10 failed authentication attempts trigger rate limiting
- **Duration**: Rate limit lasts until no failed attempts in the last 1 minute
- **Scope**: Applies to both REST API and WebSocket connections

---

### Error Responses

#### Missing Authentication
```json
{
  "error": "missing or invalid authentication"
}
```

#### Invalid Credentials (OS Auth)
```json
{
  "error": "invalid credentials"
}
```

#### Empty Command
```json
{
  "error": "command is required"
}
```

#### Rate Limited
```json
{
  "error": "Too many authentication attempts. Please try again later."
}
```

---

## WebSocket API

### Connection URL
```
ws://localhost:9091/ws
```

### Authentication

WebSocket connections use a two-step authentication process:

1. **Connect** to the WebSocket endpoint
2. **Send authentication message** with credentials within 5 seconds

#### Token Authentication
```json
{
  "type": "auth",
  "password": "YOUR_TOKEN"
}
```

#### OS Authentication
```json
{
  "type": "auth",
  "login": "username",
  "password": "password"
}
```

**Important**: The first message sent to the WebSocket must be an authentication message within 5 seconds of connection, otherwise the connection is closed with an `auth_error`.

---

### Message Types

#### Client → Server

##### `auth` - Authentication
Sent immediately after connection.

**Token-based**:
```json
{
  "type": "auth",
  "password": "YOUR_TOKEN"
}
```

**OS-based**:
```json
{
  "type": "auth",
  "login": "username",
  "password": "password"
}
```

##### `command` - Execute Command
Sends a command to the interactive shell session.

```json
{
  "type": "command",
  "command": "ls -la /tmp"
}
```

**Notes**:
- Commands are executed in an interactive shell session
- Newline is automatically appended if not present
- Multiple commands can be sent sequentially
- Command output is streamed back via `output` messages

##### `ping` - Keepalive
Sends a ping to keep the connection alive.

```json
{
  "type": "ping"
}
```

---

#### Server → Client

##### `auth_success` - Authentication Successful
Sent after successful authentication.

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

##### `auth_error` - Authentication Failed
Sent when authentication fails or times out.

```json
{
  "type": "auth_error",
  "content": "Authentication failed: empty credentials",
  "timestamp": "15:47"
}
```

**Common Errors**:
- `"Authentication failed: empty credentials"` - Missing login/password
- `"Authentication failed: invalid token"` - Invalid token provided
- `"Authentication failed: invalid credentials"` - Wrong username/password
- `"Authentication timeout: first message must be sent within 5 seconds"` - No auth sent in time
- `"Too many authentication attempts. Please try again later."` - Rate limited

##### `system` - System Message
Sent for system events (connection established, shell started, etc.).

```json
{
  "type": "system",
  "content": "Shell session started for user: token-user (session: sess-1782301685089012424)",
  "timestamp": "15:47"
}
```

##### `output` - Command Output
Sent when the shell produces output.

```json
{
  "type": "output",
  "content": "belov\n",
  "timestamp": "15:47"
}
```

**Notes**:
- Output is streamed in chunks as it becomes available
- May contain multiple lines
- Includes both stdout and stderr
- Timestamp shows when output was captured

##### `error` - Error Message
Sent when an error occurs during command execution.

```json
{
  "type": "error",
  "content": "Failed to write to shell: broken pipe",
  "timestamp": "15:47"
}
```

##### `pong` - Keepalive Response
Sent in response to a `ping` message.

```json
{
  "type": "pong",
  "content": "pong",
  "timestamp": "2026-06-24T15:47:12+04:00"
}
```

---

### WebSocket Example (Python)

```python
import asyncio
import websockets
import json

async def example():
    async with websockets.connect('ws://localhost:9091/ws', ping_interval=None) as ws:
        # Send authentication
        auth_msg = {
            'type': 'auth',
            'password': '123'  # Token
        }
        await ws.send(json.dumps(auth_msg))
        
        # Receive auth response
        response = await ws.recv()
        data = json.loads(response)
        
        if data['type'] == 'auth_success':
            print(f"Connected as {data['user']}")
            
            # Receive system message
            system_msg = await ws.recv()
            
            # Send command
            cmd = {
                'type': 'command',
                'command': 'whoami'
            }
            await ws.send(json.dumps(cmd))
            
            # Receive output
            output = await ws.recv()
            output_data = json.loads(output)
            print(f"Output: {output_data['content']}")
        else:
            print(f"Auth failed: {data['content']}")

asyncio.run(example())
```

---

## Security Considerations

### Authentication

- **Tokens** are stored in the `WASH_TOKEN` environment variable
- **OS Authentication** uses the system's authentication mechanism (PAM on Linux)
- Credentials are **not** transmitted in plaintext over HTTP (use HTTPS in production)

### Rate Limiting

- **Failed authentication attempts** are tracked per IP address
- **Threshold**: 10 failed attempts trigger rate limiting
- **Duration**: 1-minute window (resets when no failures occur)
- **Applies to**: Both REST API and WebSocket

### CORS

- **Same-origin requests** are allowed
- **Localhost** (127.0.0.1 and localhost) are allowed
- **Cross-origin requests** from other domains are rejected

### WebSocket Timeouts

- **Auth timeout**: 5 seconds (first message must be sent within this time)
- **Write timeout**: 10 seconds (per message write)

---

## Configuration

### Environment Variables

- `WASH_TOKEN` - Authentication token (loaded from `.env` or system environment)

### Command-line Flags

```bash
./WASH -os-auth -port=8080
```

- `-os-auth` - Enable OS authentication (PAM)
- `-token` - Comma-separated list of valid auth tokens
- `-allow-0` - Listen on all network interfaces (`0.0.0.0`)
- `-max-msg-size` - Maximum WebSocket message size in bytes (default: 1048576 / 1 MB)
- `-port` - Server port (default: 8080)

---

## Troubleshooting

### Connection Refused
- Ensure the server is running: `./WASH -os-auth -port=8080`
- Check the port is not in use: `lsof -i :9091`

### Authentication Failed
- **Token auth**: Verify the token in `WASH_TOKEN` environment variable
- **OS auth**: Verify username and password are correct
- Check rate limiting: Wait 1 minute if rate limited

### No Output from Commands
- Commands are executed in an interactive shell session
- Output is streamed as it becomes available (not buffered)
- Some commands may not produce immediate output (e.g., `sleep 10`)

### WebSocket Connection Closes Immediately
- Ensure first message is sent within 5 seconds
- Verify authentication message format is correct
- Check server logs for detailed error messages

---

## Limits and Constraints

| Aspect | Limit | Notes |
|--------|-------|-------|
| Command size | Unlimited | Implementation dependent |
| Output buffer | 4096 bytes per read | Streamed in chunks |
| WebSocket buffer | 4096 bytes | Read/write buffers |
| Send channel buffer | 100 messages | Prevents blocking |
| Auth timeout | 5 seconds | First message deadline |
| Write timeout | 10 seconds | Per WebSocket write |
| Rate limit threshold | 10 failures | Per IP address |
| Rate limit window | 1 minute | Sliding window |

---

## Version

- **WASH Version**: 0.1.0
- **API Version**: 1.0
- **Last Updated**: 2026-06-30
