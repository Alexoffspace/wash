# WASH Configuration

## Overview

The application supports configuration via `config.yaml` and `.env` files. This provides flexibility for different deployment scenarios while maintaining security by keeping sensitive data out of version control.

## Configuration Priority

Settings are applied in the following order (highest to lowest priority):

1. **CLI arguments** (highest priority)
2. **config.yaml**
3. **Environment variables**
4. **Default values**

## config.yaml

Create a `config.yaml` file in the application directory:

```yaml
# Enable OS authentication (true/false)
os_auth: true

# Environment variable name for the token
token: WASH_TOKEN

# Port on which the application will run
port: 9091

# Listen on 0.0.0.0 (true) or 127.0.0.1 (false)
allow_0: true

# Working directory for shell sessions and commands
# If not specified, user's home directory is used
# work_dir: /home/user/my_workspace

# Shell command for interactive sessions (e.g., bash, zsh, fish)
# If not specified or commented out, "sh" is used
# shell: bash
# On Windows use full path to shell executable:
# shell: C:\Program Files\Git\bin\bash.exe
# shell: C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe
# shell: pwsh
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `os_auth` | boolean | `false` | Enable OS user/password authentication |
| `token` | string | `""` | Environment variable name containing the auth token |
| `port` | integer | `8080` | HTTP server port |
| `allow_0` | boolean | `false` | Listen on all interfaces (0.0.0.0) instead of localhost (127.0.0.1) |
| `work_dir` | string | `""` | Working directory for shell sessions and commands (default: user's home directory) |
| `shell` | string | `sh` | Shell command for PTY sessions (e.g., bash, zsh, fish). If not set or commented out in config.yaml, defaults to `sh`. On Windows use full path: `C:\Program Files\Git\bin\bash.exe` |

## .env

Create a `.env` file to store sensitive data (tokens, passwords):

```bash
# Environment variables for WASH
WASH_TOKEN=your_token_here
```

The application automatically loads `.env` on startup if it exists.

**Important:** Never commit `.env` to version control. Add it to `.gitignore`.

## Examples

### Example 1: Development with token from environment

**config.yaml:**
```yaml
os_auth: false
token: WASH_TOKEN
port: 9091
allow_0: false
work_dir: /home/user/projects
# shell: bash
```

**.env:**
```bash
WASH_TOKEN=dev_token_123
```

**Run:**
```bash
./WASH
```

### Example 2: Production with OS authentication

**config.yaml:**
```yaml
os_auth: true
port: 9091
allow_0: true
# work_dir: /var/wash
# shell: bash
```

**Run:**
```bash
./WASH
```

### Example 3: CLI overrides config

**config.yaml:**
```yaml
os_auth: false
token: WASH_TOKEN
port: 9091
allow_0: false
work_dir: /home/user/projects
# shell: zsh
```

**Run with CLI override:**
```bash
./WASH -os-auth -allow-0
```

Result: OS authentication enabled and listening on 0.0.0.0 (config values for these options are overridden). `work_dir` and `shell` are still read from config.yaml.

## Security Considerations

1. **Protect .env file:** Ensure proper file permissions (e.g., `chmod 600 .env`) and don't commit it to version control.
2. **Use strong tokens:** Generate long, random tokens for authentication.
3. **Limit network exposure:** Use `allow_0: false` (default) to listen only on localhost unless you need external access.
4. **Use HTTPS in production:** The application doesn't include TLS — use a reverse proxy (nginx, Caddy) in front.
5. **Restrict CORS:** Configure `CheckOrigin` in production to allow only trusted origins.

## Migration from CLI-only

If you're currently using only CLI arguments, you can migrate to config.yaml:

1. Create `config.yaml` with your desired settings
2. Move sensitive tokens to `.env` and reference them in `config.yaml`
3. Remove CLI arguments and run `./WASH`
4. Verify behavior matches your previous CLI-based setup

Alternatively, use the interactive setup scripts (`setup-linux.sh` / `setup-windows.ps1`) which guide you through configuration in a TUI and generate `config.yaml` and `.env` automatically.

## Troubleshooting

**Config not loading:**
- Check that `config.yaml` is in the same directory as the binary
- Verify YAML syntax (use a YAML linter)
- Check application logs for configuration loading messages

**Environment variable not found:**
- Ensure the variable name in `config.yaml` matches the `.env` file
- Check that `.env` is loaded before the application starts
- Verify the variable is set in the environment
