# Contributing to WASH

Thank you for your interest in contributing to WASH! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)
- [Community Guidelines](#community-guidelines)

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git
- A modern web browser (for testing the web UI)

### Setting Up the Development Environment

1. **Fork the repository**
   ```bash
    git clone https://github.com/your-username/WASH.git
    cd WASH
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build the project**
   ```bash
   go build -o WASH
   ```

4. **Run tests**
   ```bash
   go test -v ./...
   ```

5. **Run the server**
   ```bash
   ./WASH -token=test -port=9091
   ```

## Development Workflow

1. Create a new branch for your feature or bugfix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and test thoroughly.

3. Run all tests and ensure they pass:
   ```bash
   go test -v -race ./...
   ```

4. Commit your changes (see [Commit Messages](#commit-messages)).

5. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Open a Pull Request.

## Code Style

### Go Code

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` or `goimports` for formatting
- Run `go vet` before committing:
  ```bash
  go vet ./...
  ```
- Ensure 100% test coverage for new features

### JavaScript/HTML/CSS

- Use semantic HTML5 elements
- Follow BEM naming convention for CSS classes
- Use ES6+ JavaScript features
- Add comments in English only
- Keep code DRY (Don't Repeat Yourself)

### General

- All comments must be in English
- Use meaningful variable and function names
- Keep functions small and focused
- Add error handling for all external calls

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific test
go test -v -run TestShellSession ./...

# Run with race detector
go test -v -race ./...
```

### Writing Tests

- Write tests for all new features
- Use table-driven tests where appropriate
- Test both success and error cases
- Include benchmarks for performance-critical code

Example:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"valid input", "test", "test"},
        {"empty input", "", ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Process(tt.input)
            if got != tt.want {
                t.Errorf("Process(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples

```
feat(api): add health check endpoint
fix(ws): resolve race condition in session cleanup
docs: update installation instructions
refactor(auth): extract token validation logic
test: add integration tests for REST API
```

## Pull Request Process

1. **Ensure your PR description is clear**
   - Describe the changes
   - Link related issues
   - Provide testing instructions

2. **Update documentation**
   - Update README if needed
   - Add examples for new features
   - Document API changes

3. **Ensure CI passes**
   - All tests must pass
   - Code must be formatted
   - No linting errors

4. **Request review**
   - Tag maintainers for review
   - Address feedback promptly

5. **Merge**
   - Squash commits if necessary
   - Update version in CHANGELOG

## Documentation

### Adding New Documentation

- Place new documentation in the `docs/` directory
- Keep documentation in sync with code changes
- Use Markdown format
- Include examples where applicable

### Documentation Structure

```
docs/
├── README.md              # English documentation
├── INDEX.md               # Quick start index
├── API_REFERENCE.md       # API reference
├── AUTO_TESTS.md          # Auto-tests guide (Russian)
├── CONFIGURATION.md       # Configuration guide
├── ENDPOINTS_SUMMARY.md   # Endpoint summary
├── TESTING.md             # Test results
└── TROUBLESHOOTING.md     # Troubleshooting guide
```

All configuration lives in `config.yaml` (YAML) or `.env` (environment variables).

## Community Guidelines

### Be Respectful

- Treat everyone with respect
- Provide constructive feedback
- Accept constructive criticism

### Be Collaborative

- Help others with their issues
- Share knowledge and best practices
- Review pull requests promptly

### Be Clear

- Ask questions if something is unclear
- Provide context in discussions
- Use clear and concise language

## Questions?

If you have questions or need help:

- Open an issue for bugs
- Start a discussion for feature requests
- Contact the maintainers directly

## Thank You!

Your contributions make WASH better for everyone. Thank you for taking the time to contribute!
