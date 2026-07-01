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

### Planned
- HTTPS/TLS support (requires reverse proxy configuration)
- Role-based access control (RBAC)
- Command logging and audit trail
- Multi-user session sharing
- File upload/download via WebSocket
- Command autocomplete suggestions
- Syntax highlighting for output
- Keyboard shortcuts documentation
- Docker container support
- Systemd service configuration
- Windows service installation
