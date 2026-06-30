# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-24

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
