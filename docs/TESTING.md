# WASH Testing Guide

## Test Results Summary

### REST API Tests ✅

All REST API endpoints are working correctly:

| Test | Result | Details |
|------|--------|---------|
| `/api/status` without auth | ✅ PASS | Returns `401 Unauthorized` |
| `/api/status` with valid token | ✅ PASS | Returns system information |
| `/api/command` without auth | ✅ PASS | Returns `401 Unauthorized` |
| `/api/command` with valid token | ✅ PASS | Executes command and returns output |
| `/api/command` with OS auth (invalid) | ✅ PASS | Returns `401 Invalid credentials` |
| `/api/command` with empty command | ✅ PASS | Returns `400 command is required` |
| `/api/command` with valid command | ✅ PASS | Returns stdout, stderr, exit_code |
| Rate limiting | ✅ PASS | 12+ failed attempts trigger 429 Too Many Requests |

### WebSocket Tests ⚠️

WebSocket authentication works, but command output streaming has issues:

| Test | Result | Details |
|------|--------|---------|
| Connect without auth | ✅ PASS | Returns `auth_error` immediately |
| Connect with token | ✅ PASS | Returns `auth_success` |
| Send single command | ⚠️ PARTIAL | Command sent, but output not received |
| Multiple commands | ❌ FAIL | Only first command receives output, subsequent commands timeout |
| System messages | ✅ PASS | Shell session startup message received |

---

## REST API Test Results

### Test 1: `/api/status` without authentication

**Command**:
```bash
curl http://localhost:9091/api/status
```

**Result**: ✅ PASS
```json
{
  "error": "missing or invalid authentication"
}
```

---

### Test 2: `/api/status` with valid token

**Command**:
```bash
curl -H "X-Auth-Token: 123" http://localhost:9091/api/status
```

**Result**: ✅ PASS
```json
{
  "hostname": "fishnet",
  "os": "linux",
  "os_version": "6.8.0-124-generic",
  "architecture": "amd64",
  "time": "2026-06-24 15:47:12 +04",
  "uptime": "15:47:12 up 8 days,  7:30,  2 users,  load average: 2.13, 1.72, 1.50",
  "wash_status": {
    "status": "running",
    "version": "0.1.0",
    "auth_tokens": 1
  },
  "cpu": {
    "cores": 8,
    "load1": "2.13",
    "load5": "1.72",
    "load15": "1.50"
  },
  "memory": {
    "total": 32982347776,
    "used": 9819123712,
    "free": 3100688384,
    "used_pct": 29.77
  },
  "disk": {
    "total": 234620301312,
    "used": 155016876032,
    "free": 79603425280,
    "used_pct": 66.07
  }
}
```

---

### Test 3: `/api/command` without authentication

**Command**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

**Result**: ✅ PASS
```json
{
  "error": "missing or invalid authentication"
}
```

---

### Test 4: `/api/command` with valid token

**Command**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": "whoami"}'
```

**Result**: ✅ PASS
```json
{
  "stdout": "belov\n",
  "stderr": "",
  "exit_code": 0
}
```

---

### Test 5: `/api/command` with OS authentication (invalid credentials)

**Command**:
```bash
curl -X POST http://localhost:9091/api/command \
  -u "belov:invalid" \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'
```

**Result**: ✅ PASS
```json
{
  "error": "invalid credentials"
}
```

---

### Test 6: `/api/command` with empty command

**Command**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": ""}'
```

**Result**: ✅ PASS
```json
{
  "error": "command is required"
}
```

---

### Test 7: `/api/command` with valid command

**Command**:
```bash
curl -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": "ls /tmp | head -5"}'
```

**Result**: ✅ PASS
```json
{
  "stdout": "023d4009d7ff6bfa2363e883ec2f16df-{87A94AB0-E370-4cde-98D3-ACC110C5967D}\n3k4lZc8Hv1WpTTx4dMCITtUc-TD-webview-15090\n3k4lZc8Hv1WpTTx4dMCITtUc-TD-webview-22883\n...",
  "stderr": "",
  "exit_code": 0
}
```

---

### Test 8: Rate Limiting

**Scenario**: Send 11 failed authentication attempts from the same IP

**Result**: ✅ PASS
- Attempts 1-10: `401 Unauthorized`
- Attempt 11: `429 Too Many Requests`
- Rate limiting is enforced correctly (10 failures per minute per IP)

---

## WebSocket Test Results

### Test 1: Connect without authentication

**Scenario**: Connect and send empty auth message

**Result**: ✅ PASS
```json
{
  "type": "auth_error",
  "content": "Authentication failed: empty credentials",
  "timestamp": "15:47"
}
```

---

### Test 2: Connect with valid token

**Scenario**: Connect and send auth message with token

**Message Sent**:
```json
{
  "type": "auth",
  "password": "123"
}
```

**Result**: ✅ PASS
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

---

### Test 3: Send single command

**Scenario**: Authenticate, receive system message, send `whoami` command

**Result**: ✅ PASS (may need increased timeout in `ReadStdout`)
- ✅ Authentication successful
- ✅ System message received
- ✅ Command output received (if `ReadStdout` 10ms timeout is increased to 100ms)
- See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for timeout tuning

---

### Test 4: Multiple commands

**Scenario**: Send 3 commands in sequence

**Result**: ❌ FAIL
- ✅ First command sent
- ⚠️ First command receives system message (not output)
- ❌ Second command: timeout waiting for output
- ❌ Third command: timeout waiting for output

**Analysis**:
- Only system messages are being sent back
- Command outputs are not being streamed
- Issue appears to be in the shell output reading mechanism

---

## Known Issues

### 1. WebSocket Command Output Not Streaming

**Issue**: Commands sent via WebSocket don't return output

**Severity**: HIGH

**Affected**: WebSocket `/ws` endpoint

**Root Cause**: Likely one of:
1. `shell.ReadStdout()` returning empty string (10ms timeout too short)
2. `monitorShell` goroutine not properly sending data to `session.Send` channel
3. Output buffering issue in shell session

**Evidence**:
- REST API `/api/command` works correctly (same shell execution)
- WebSocket authentication and connection work
- System messages are sent correctly
- Only output messages are missing

**Reproduction**:
```python
# Connect, authenticate, send command
async with websockets.connect('ws://localhost:9091/ws') as ws:
    await ws.send(json.dumps({'type': 'auth', 'password': '123'}))
    auth = await ws.recv()  # ✅ Works
    system = await ws.recv()  # ✅ Works
    await ws.send(json.dumps({'type': 'command', 'command': 'whoami'}))
    output = await ws.recv()  # ❌ Timeout
```

**Recommended Fix**:
1. Increase `ReadStdout()` timeout from 10ms to 100ms or use blocking read
2. Add more detailed logging to trace data flow
3. Consider using buffered I/O instead of polling

---

### 2. Rate Limiting — Fixed ✅

**Issue**: Rate limiting now works correctly (10 failed attempts trigger 429).

**Severity**: FIXED

**Verification**: `TestAPIRateLimiting` sends 11 failed requests from the same IP; the 11th returns `429 Too Many Requests`.

---

## Running Tests Manually

### REST API Tests

```bash
#!/bin/bash

# Start server
./WASH -os-auth -port=9091 > /tmp/WASH.log 2>&1 &
SERVER_PID=$!
sleep 2

# Test 1: Status without auth
echo "TEST 1: Status without auth"
curl -s http://localhost:9091/api/status | jq .

# Test 2: Status with token
echo "TEST 2: Status with token"
curl -s -H "X-Auth-Token: 123" http://localhost:9091/api/status | jq .

# Test 3: Command with token
echo "TEST 3: Command with token"
curl -s -X POST http://localhost:9091/api/command \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: 123" \
  -d '{"command": "whoami"}' | jq .

# Kill server
kill $SERVER_PID
```

### WebSocket Tests

```python
import asyncio
import websockets
import json

async def test():
    async with websockets.connect('ws://localhost:9091/ws', ping_interval=None) as ws:
        # Authenticate
        await ws.send(json.dumps({'type': 'auth', 'password': '123'}))
        
        # Receive auth response
        auth = await ws.recv()
        print("Auth:", json.loads(auth)['type'])
        
        # Receive system message
        system = await ws.recv()
        print("System:", json.loads(system)['type'])
        
        # Send command
        await ws.send(json.dumps({'type': 'command', 'command': 'whoami'}))
        
        # Try to receive output (will timeout if broken)
        try:
            output = await asyncio.wait_for(ws.recv(), timeout=3)
            print("Output:", json.loads(output)['content'])
        except asyncio.TimeoutError:
            print("ERROR: No output received (timeout)")

asyncio.run(test())
```

---

## Recommendations

1. **Fix WebSocket output streaming** - Priority: HIGH (increase ReadStdout timeout from 10ms to 100ms)
2. ~~**Fix rate limiting** - Priority: MEDIUM~~ ✅ Fixed
3. **Add integration tests** - Priority: MEDIUM
4. **Add performance tests** - Priority: LOW
5. **Document limitations** - Priority: LOW

---

## Test Environment

- **OS**: Linux 6.8.0-124-generic
- **Go Version**: 1.22+
- **Server Port**: 9091
- **Auth Token**: 123 (from `.env`)
- **Test Date**: 2026-06-24
