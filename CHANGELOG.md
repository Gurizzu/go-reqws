# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of go-reqws
- HTTP client with functional options pattern
- WebSocket streaming with channel-based communication
- Automatic retry mechanism with exponential backoff
- WebSocket auto-reconnection support
- Middleware/hooks system (before/after/error hooks)
- Custom error types (HTTPError, WebSocketError)
- Response helper methods (JSON, String, IsSuccess, etc.)
- Logger interface for custom logging integration
- Secure TLS by default with opt-in insecure mode
- Context support for all operations
- Comprehensive documentation and examples

### Security
- Removed hardcoded `InsecureSkipVerify: true`
- TLS certificate verification enabled by default
- Opt-in insecure mode via `WithInsecureSkipVerify()` option

### Changed
- Package renamed from `main` to `reqws` for library usage
- Module path updated to `github.com/gurizzu/go-reqws`
- Removed hardcoded logging with aurora dependency
- Refactored duplicate code in request methods (DRY principle)

### Fixed
- Form fields now properly handled in `NewRequestWithResponse()`
- Connection leaks prevented in retry logic
- Proper cleanup of response bodies

## [0.1.0] - TBD

Initial release.

### Features

#### HTTP Client
- Functional options pattern for clean API
- Context-aware requests with timeout/cancellation support
- Multiple HTTP methods (GET, POST, PUT, DELETE, etc.)
- Query parameter support (single and bulk)
- JSON request body marshaling
- Multipart file uploads with form data
- Custom headers and authentication
- Two response modes: body-only or full response

#### WebSocket
- Bidirectional streaming with channels
- JSON message serialization
- Message compression support (1MB read limit)
- Context-based cancellation
- Graceful connection closure

#### Retry Mechanism
- Exponential backoff algorithm
- Smart retry logic (retry 5xx, 429, network errors; skip 4xx)
- Configurable max retries, delays, and multiplier
- Default configuration for quick setup
- Context-aware with proper cancellation

#### WebSocket Auto-Reconnection
- Automatic reconnection on connection drop
- Exponential backoff for reconnection delays
- Configurable max reconnection attempts
- Optional callback on each reconnection attempt
- Logger integration for tracking attempts

#### Middleware/Hooks
- BeforeRequest hooks for request modification
- AfterResponse hooks for response processing
- OnError hooks for error handling/logging
- Multiple hooks support (executed in order)
- Use cases: logging, metrics, tracing, custom auth

#### Error Handling
- `HTTPError` type for non-2xx HTTP responses
  - Includes status code and response body
  - Implements standard error interface
- `WebSocketError` type for WebSocket failures
  - Includes reason and underlying error
  - Supports error unwrapping for error chains

#### Response Helpers
- `JSON(v interface{})` - Auto unmarshal response body
- `String()` - Get response body as string
- `IsSuccess()` - Check for 2xx status
- `IsClientError()` - Check for 4xx status
- `IsServerError()` - Check for 5xx status

#### Logging
- Logger interface for custom integration
- Support for slog, zap, logrus, and custom loggers
- Silent by default (no forced logging)
- Optional debug/info/error logging throughout

#### Security
- Secure TLS certificate verification by default
- Optional `WithInsecureSkipVerify()` for testing
- No hardcoded credentials or secrets
- Proper context handling to prevent leaks

### Dependencies
- `github.com/coder/websocket` v1.8.14 - WebSocket implementation

---

## Release Notes

### v0.1.0 - Initial Release

This is the first release of `go-reqws`, a lightweight HTTP client library with built-in WebSocket streaming support, designed for Go microservices and real-time applications.

**Key Features:**
- Clean, idiomatic Go API using functional options pattern
- Built-in retry mechanism with smart exponential backoff
- WebSocket auto-reconnection for resilient real-time connections
- Extensible middleware/hooks system for logging, metrics, and custom logic
- Type-safe error handling with custom error types
- Convenient response helpers for common operations
- Production-ready with secure defaults

**Use Cases:**
- Microservices needing both HTTP and WebSocket communication
- Real-time applications (chat, notifications, live updates)
- API client libraries
- Projects requiring simplicity without sacrificing features

**Getting Started:**
```bash
go get github.com/gurizzu/go-reqws
```

See [README.md](README.md) for complete documentation and examples.

---

## Upgrade Guide

### From Pre-release to v0.1.0

If you were using the pre-release version with `package main`, you need to:

1. **Update import path:**
   ```go
   // Old
   import "go-reqws"

   // New
   import "github.com/gurizzu/go-reqws"
   ```

2. **Update module:**
   ```bash
   go get github.com/gurizzu/go-reqws@latest
   ```

3. **Security update:**
   - WebSocket TLS verification is now **enabled by default**
   - If you need insecure mode for testing, explicitly add:
     ```go
     reqws.WithInsecureSkipVerify()
     ```

4. **Logging changes:**
   - No more forced colored output
   - Logging is silent by default
   - To enable logging, provide a logger:
     ```go
     client.WithLogger(yourLogger)
     ```

5. **Error handling:**
   - Use type assertions for better error handling:
     ```go
     var httpErr *reqws.HTTPError
     if errors.As(err, &httpErr) {
         // Handle HTTP error with status code and body
     }
     ```

---

## Future Roadmap

### v0.2.0 (Planned)
- [ ] HTTP/2 support
- [ ] Request/response compression
- [ ] Circuit breaker pattern
- [ ] Rate limiting
- [ ] Request mocking for testing
- [ ] More examples and tutorials

### v1.0.0 (Planned)
- [ ] Stable API guarantee
- [ ] Performance benchmarks
- [ ] Comprehensive test coverage (>90%)
- [ ] Production usage validation
- [ ] Community feedback integration

---

[Unreleased]: https://github.com/gurizzu/go-reqws/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/gurizzu/go-reqws/releases/tag/v0.1.0
