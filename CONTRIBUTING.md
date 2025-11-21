# Contributing to go-reqws

First off, thank you for considering contributing to go-reqws! It's people like you that make go-reqws such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

**Bug Report Template:**

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Create client with '...'
2. Call method '....'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Code snippet**
```go
// Minimal reproducible example
```

**Environment:**
- Go version: [e.g., 1.22.1]
- go-reqws version: [e.g., v0.1.0]
- OS: [e.g., Ubuntu 22.04]

**Additional context**
Add any other context about the problem here.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- **Clear title and description** of the proposed feature
- **Use cases** - why would this feature be useful?
- **Possible implementation** - if you have ideas on how to implement it
- **Alternatives considered** - what other solutions did you think about?

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Add tests** if you've added code that should be tested
4. **Update documentation** if you've changed APIs
5. **Ensure the test suite passes** (`go test ./...`)
6. **Format your code** (`go fmt ./...`)
7. **Submit the pull request**

## Development Setup

### Prerequisites

- Go 1.22 or higher
- Git
- Basic understanding of Go and HTTP/WebSocket protocols

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/go-reqws.git
cd go-reqws

# Add upstream remote
git remote add upstream https://github.com/gurizzu/go-reqws.git

# Create a branch for your changes
git checkout -b feature/your-feature-name

# Make your changes and commit
git add .
git commit -m "feat: add awesome feature"

# Push to your fork
git push origin feature/your-feature-name

# Create a pull request on GitHub
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific tests
go test -v -run TestNewRequest

# Run tests with race detector
go test -v -race ./...
```

### Code Style

We follow standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter (recommended: golangci-lint)
golangci-lint run

# Check for common issues
go vet ./...
```

**Code Style Guidelines:**

1. **Use `gofmt`** - all code must be formatted
2. **Follow Go idioms** - read [Effective Go](https://golang.org/doc/effective_go.html)
3. **Write godoc comments** - for all exported functions, types, and constants
4. **Keep functions small** - prefer small, focused functions
5. **Error handling** - always handle errors, never ignore them
6. **Use context** - always accept `context.Context` for operations that can block

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): subject

body (optional)

footer (optional)
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

**Examples:**

```
feat(retry): add custom retry predicate function

Allow users to provide custom logic to determine if a request
should be retried, giving more control over retry behavior.

Closes #42
```

```
fix(websocket): prevent connection leak on reconnection failure

Properly close WebSocket connections that fail to reconnect
to avoid resource leaks.

Fixes #55
```

### Testing Guidelines

1. **Write tests for new features**
   - Unit tests for individual functions
   - Integration tests for end-to-end scenarios

2. **Maintain high coverage**
   - Aim for >80% code coverage
   - Cover edge cases and error paths

3. **Use table-driven tests** when appropriate:

```go
func TestWithQueryParam(t *testing.T) {
    tests := []struct {
        name     string
        key      string
        value    string
        expected string
    }{
        {"simple param", "foo", "bar", "foo=bar"},
        {"with spaces", "name", "John Doe", "name=John+Doe"},
        {"special chars", "email", "test@example.com", "email=test%40example.com"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

4. **Mock external dependencies**
   - Use `httptest` for HTTP servers
   - Mock WebSocket connections when appropriate

### Documentation Guidelines

1. **Godoc comments** for all exported symbols:

```go
// NewRequests creates a new HTTP client with the specified base URL and timeout.
//
// The baseURL should not include a trailing slash. All request paths will be
// appended to this base URL.
//
// Example:
//
//	client := reqws.NewRequests("https://api.example.com", 30*time.Second)
//	body, err := client.NewRequest(ctx, reqws.WithPath("/users"))
func NewRequests(baseURL string, timeout time.Duration) *Requests {
    // ...
}
```

2. **Update README.md** when adding features
3. **Add examples** in godoc and README
4. **Update CHANGELOG.md** for notable changes

### Project Structure

```
go-reqws/
├── requests.go        # HTTP client implementation
├── ws.go              # WebSocket implementation
├── retry.go           # Retry mechanism
├── middleware.go      # Hooks/middleware system
├── errors.go          # Custom error types
├── *_test.go          # Test files
├── examples/          # Example code (planned)
└── docs/              # Additional documentation (planned)
```

## What to Work On?

### Good First Issues

Look for issues labeled `good first issue` - these are suitable for newcomers.

### Help Wanted

Issues labeled `help wanted` need community contributions.

### Feature Roadmap

Check [PLANNING.md](PLANNING.md) or [CHANGELOG.md](CHANGELOG.md) for planned features.

## Review Process

1. **Automated checks** must pass (tests, linting)
2. **Code review** by maintainers
3. **Discussion** of design decisions if needed
4. **Approval** and merge by maintainer

**Review timeline:**
- We aim to review PRs within 3-5 days
- Complex PRs may take longer
- Feel free to ping if no response after 1 week

## Questions?

- **General questions:** Open a [GitHub Discussion](https://github.com/gurizzu/go-reqws/discussions)
- **Bug reports:** Open a [GitHub Issue](https://github.com/gurizzu/go-reqws/issues)
- **Feature requests:** Open a [GitHub Issue](https://github.com/gurizzu/go-reqws/issues) with the `enhancement` label

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- CHANGELOG.md for significant contributions
- README.md (if applicable)

---

Thank you for contributing to go-reqws! 🎉
