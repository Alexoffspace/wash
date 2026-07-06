# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-06-30

### Added
- xterm.js terminal with `fit` addon for proper terminal emulation
- 10 background images for dark and light themes
- MesloLGM Nerd Font (Bold, Bold Italic, Italic, Regular) with SIL OFL license
- `backgrounds` configuration section in `config.yaml` for theme backgrounds

### Changed
- Replaced custom terminal UI with full xterm.js integration
- Major HTML template refactoring (982 lines diff) — cleaner markup, dynamic theme switching
- PTY output handling in `pkg/shell/shell.go` — improved read loop with configurable timeout
- WebSocket session lifecycle in `pkg/ws/ws.go` — better resize handling and origin validation
- Updated all documentation (API reference, endpoints summary, testing guide, troubleshooting)
- Updated `integration_test.go` — added tests for xterm.js endpoint and theme backgrounds

### Infrastructure
- Added static assets: `static/xterm.js`, `static/xterm.css`, `static/xterm-addon-fit.js`
- Added `static/backgrounds/` with themed images (dark/light)
- Added `static/fonts/` with MesloLGM Nerd Font files

## [1.0.0] - 2026-06-30

### Added
- Initial release
- WebSocket interface for interactive shell sessions
- REST API for single-command execution
- Token-based authentication
- OS user/password authentication (Linux `su`, Windows `net user`)
- Web-based terminal UI with dark/light theme
- Graceful shutdown on SIGINT/SIGTERM
- Automatic ping/keep-alive mechanism (30s interval)
- Multiple concurrent session support
- Command history navigation with arrow keys
- Tab-based word autocompletion
- Copy-to-clipboard functionality for all message blocks
- Real-time output streaming
- Session management with cleanup
- Health check endpoint (`GET /api/status`)

### Security
- Token authentication with configurable tokens
- HTTP Basic Auth support for OS authentication
- CORS configuration with origin validation (loopback/localhost in default mode; host-matching in `-allow-0` mode)
- Input validation for shell commands
- Maximum message size limit (1MB default)

### Infrastructure
- Cross-platform build support (Linux, macOS, Windows)
- Go module configuration
- Integration tests
- Benchmarks for performance testing

## [0.1.0] - 2026-06-30

### Added
- Initial release
- WebSocket interface for interactive shell sessions
- REST API for single-command execution
- Token-based authentication
- OS user/password authentication (Linux `su`, Windows `net user`)
- Web-based terminal UI with dark/light theme
- Graceful shutdown on SIGINT/SIGTERM
- Automatic ping/keep-alive mechanism (30s interval)
- Multiple concurrent session support
- Command history navigation with arrow keys
- Tab-based word autocompletion
- Copy-to-clipboard functionality for all message blocks
- Real-time output streaming
- Session management with cleanup
- Health check endpoint (`GET /api/status`)

### Security
- Token authentication with configurable tokens
- HTTP Basic Auth support for OS authentication
- CORS configuration with origin validation (loopback/localhost in default mode; host-matching in `-allow-0` mode)
- Input validation for shell commands
- Maximum message size limit (1MB default)

### Infrastructure
- Cross-platform build support (Linux, macOS, Windows)
- Go module configuration
- Integration tests
- Benchmarks for performance testing

## [Unreleased]

### Added
- Windows ConPTY support (`CreatePseudoConsole`) with UTF-16↔UTF-8 conversion and OEM/ANSI codepage decoding
- Real CPU usage on Windows via WMI (`Win32_Processor`)
- Real system memory on Windows via WMI (`Win32_OperatingSystem`)
- Real system uptime on Windows (seconds → readable format)
- macOS memory info via `sysctl` + `vm_stat`
- BOM (UTF-8 BOM) stripping in `config.yaml` and `.env` loading
- `windowsMode: true` in xterm.js for correct Ctrl+C/V behavior on Windows
- Unit tests for API package (`pkg/api/api_test.go`)
- Integration test validation for `/api/status` JSON response fields
- Setup scripts: `Enable token authentication?` prompt — allows disabling token auth entirely (token: "" in config)

### Changed
- `go.mod`: Go version bumped to 1.25; added `golang.org/x/sys` and `golang.org/x/text` dependencies
- `pkg/shell/shell.go`: `RunCommand` uses `-Command` flag on Windows and decodes OEM codepage output
- `pkg/shell/session_windows.go`: Complete rewrite — ConPTY support with pipe fallback, line editing, environment setup
- `pkg/api/api.go`: Cross-platform metrics — Windows CPU/memory/uptime via WMI, macOS memory via sysctl+vm_stat
- `pkg/config/config.go`: BOM-safe YAML and .env parsing
- `setup-linux.sh` / `setup-windows.ps1`: Removed service installation (now build-only scripts)

### Infrastructure
- Added `golang.org/x/sys v0.46.0` for Windows API calls
- Added `golang.org/x/text v0.38.0` for OEM/ANSI codepage decoding
