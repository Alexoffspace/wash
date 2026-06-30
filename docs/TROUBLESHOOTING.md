# WASH Troubleshooting Guide

## Common Issues

### 1. WebSocket Commands Don't Return Output

**Symptom**: 
- WebSocket connection works (auth successful)
- Commands are sent without error
- No output is received (timeout)
- REST API `/api/command` works fine

**Root Cause**: 
The `monitorShell` goroutine uses a 10ms timeout when reading shell output, which is too short. Output may not be available in that window, resulting in empty reads.

**Solution**:

Edit `pkg/shell/shell.go` and increase the timeout in `ReadStdout()`:

```go
// Before (line 104):
case <-time.After(10 * time.Millisecond):

// After:
case <-time.After(100 * time.Millisecond):
```

Then rebuild and restart:
```bash
go build -o WASH
./WASH -os-auth -port=9091
```

**Alternative**: Use blocking read instead of polling (requires refactoring):
```go
// Use io.ReadAll or bufio.Scanner for blocking reads
```

---

### 2. ~~Rate Limiting Not Working~~ ✅ Fixed

**Status**: Fixed. Rate limiting now correctly blocks after 10 failed attempts per minute per IP.

**Verification**: Run `go test -v -run TestAPIRateLimiting .`

---

### 3. Server Won't Start

**Symptom**:
```
bind: address already in use
```

**Solution**:
```bash
# Kill existing process
pkill -9 WASH

# Or use a different port
./WASH -os-auth -port=9092
```

Check what's using the port:
```bash
lsof -i :9091
netstat -tlnp | grep 9091
```

---

### 4. Authentication Fails

**Token Auth**:
```bash
# Verify token in .env
cat .env

# Make sure token is passed correctly
curl -H "X-Auth-Token: YOUR_TOKEN" http://localhost:9091/api/status
```

**OS Auth**:
```bash
# Test OS authentication manually
su - username

# Verify user exists
id username

# Check if OS auth is enabled
./WASH -os-auth -port=9091
```

---

### 5. CORS Errors in Browser

**Symptom**:
```
Access to XMLHttpRequest at 'http://localhost:9091/ws' from origin 'http://example.com' 
has been blocked by CORS policy
```

**Solution**:
- For local development: Use `http://localhost:9091` in browser (loopback origins allowed by default)
- For production with `-allow-0`: Origin must match the request Host. Ensure `X-Forwarded-Host` is set correctly behind a reverse proxy.
- The CORS logic is in `pkg/ws/ws.go` (`isAllowedOrigin` function, lines 35-59):
  - In localhost-only mode: only loopback origins (`localhost`, `127.0.0.1` etc.) are accepted
  - In `-allow-0` mode: the origin must match the request's Host header exactly
  - Non-browser clients (no `Origin` header) are allowed through; authentication is still required

---

### 6. WebSocket Connection Closes Immediately

**Symptom**:
```
WebSocket connection closed
```

**Causes**:
1. First message not sent within 5 seconds
2. Invalid authentication message format
3. Server crashed

**Solution**:
```bash
# Check server logs
tail -f /tmp/WASH.log

# Verify auth message format
# Must be valid JSON with 'type': 'auth'
{
  "type": "auth",
  "password": "YOUR_TOKEN"
}
```

---

### 7. High Memory Usage

**Symptom**:
- Server memory usage grows over time
- Memory not released after sessions close

**Causes**:
1. Sessions not properly cleaned up
2. Goroutine leaks
3. Channel not closed

**Solution**:
```bash
# Monitor memory usage
watch -n 1 'ps aux | grep WASH'

# Check goroutine count
curl http://localhost:9091/api/status | jq .

# Restart server if needed
pkill -9 WASH
./WASH -os-auth -port=9091
```

---

### 8. Slow Command Execution

**Symptom**:
- Commands take longer than expected
- Output arrives in chunks with delays

**Causes**:
1. Shell session startup overhead
2. Network latency
3. System load

**Solution**:
- Use REST API for one-off commands (faster)
- Keep WebSocket connection open for multiple commands
- Check system load: `uptime`

---

### 9. Commands with Special Characters Fail

**Symptom**:
```json
{
  "error": "command execution failed"
}
```

**Solution**:
Properly escape JSON strings:
```bash
# Before (wrong):
curl -X POST http://localhost:9091/api/command \
  -d '{"command": "echo "hello""}'

# After (correct):
curl -X POST http://localhost:9091/api/command \
  -d '{"command": "echo \"hello\""}'

# Or use jq:
curl -X POST http://localhost:9091/api/command \
  -d "$(jq -n --arg cmd 'echo "hello"' '{command: $cmd}')"
```

---

### 10. WebSocket Ping/Pong Issues

**Symptom**:
- Connection drops after inactivity
- Ping messages not received

**Solution**:
The server sends ping every 30 seconds. Client should respond with pong:
```python
async def handle_ping(ws):
    async for message in ws:
        data = json.loads(message)
        if data['type'] == 'ping':
            await ws.send(json.dumps({'type': 'pong'}))
```

---

## Debugging

### Enable Verbose Logging

The server logs to stdout. Capture logs:
```bash
./WASH -os-auth -port=9091 2>&1 | tee /tmp/WASH.log
```

Look for:
- `[SESSION ...]` messages for WebSocket sessions
- `shell:` messages for shell operations
- `WebSocket` messages for connection issues

### Test with curl

```bash
# Test REST API
curl -v -H "X-Auth-Token: 123" http://localhost:9091/api/status

# Test WebSocket (requires wscat or similar)
wscat -c ws://localhost:9091/ws
```

### Test with Python

```python
import asyncio
import websockets
import json

async def test():
    async with websockets.connect('ws://localhost:9091/ws', ping_interval=None) as ws:
        # Send auth
        await ws.send(json.dumps({'type': 'auth', 'password': '123'}))
        
        # Check response
        resp = await ws.recv()
        print("Auth response:", json.loads(resp))
        
        # Send command
        await ws.send(json.dumps({'type': 'command', 'command': 'echo test'}))
        
        # Wait for output
        try:
            output = await asyncio.wait_for(ws.recv(), timeout=5)
            print("Output:", json.loads(output))
        except asyncio.TimeoutError:
            print("ERROR: No output received")

asyncio.run(test())
```

---

## Performance Tuning

### Increase Buffer Sizes

In `pkg/ws/ws.go` (line 88):
```go
// Before:
Send: make(chan []byte, 100)

// After (for high throughput):
Send: make(chan []byte, 1000)
```

### Increase Shell Output Read Timeout

In `pkg/shell/shell.go` (line 104):
```go
// Before:
case <-time.After(10 * time.Millisecond):

// After:
case <-time.After(500 * time.Millisecond):
```

### Increase Max Message Size

Command-line flag:
```bash
./WASH -os-auth -port=9091 -max-msg-size=10485760  # 10 MB
```

---

## Getting Help

1. **Check logs**: `tail -f /tmp/WASH.log`
2. **Review API docs**: See `docs/API_REFERENCE.md`
3. **Check test results**: See `docs/TESTING.md`
4. **Enable debug logging**: Add `log.Printf()` calls to source code

---

## Known Limitations

1. **WebSocket output streaming** - 10ms timeout may miss early output (see issue #1)
2. ~~**Rate limiting** - Not enforced~~ ✅ Fixed
3. **Windows OS auth** - Only checks user existence, not password
4. **Session limits** - No built-in limit on concurrent sessions
5. **Input validation** - Commands executed directly (no sanitization)

---

## Related Documentation

- [API_REFERENCE.md](API_REFERENCE.md) - Complete API documentation
- [TESTING.md](TESTING.md) - Test results and known issues
- [CONFIGURATION.md](CONFIGURATION.md) - Configuration guide
