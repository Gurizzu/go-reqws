# go-reqws

> Simple HTTP client with built-in WebSocket streaming for Go microservices

[![Go Reference](https://pkg.go.dev/badge/github.com/gurizzu/go-reqws.svg)](https://pkg.go.dev/github.com/gurizzu/go-reqws)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Features

- ✅ **Clean functional options pattern** - Idiomatic Go API design
- ✅ **Context-first design** - Proper cancellation and timeout support
- ✅ **Built-in WebSocket streaming** - Bidirectional communication with channels
- ✅ **Automatic retry with exponential backoff** - Smart retry logic for transient failures
- ✅ **WebSocket auto-reconnection** - Resilient real-time connections
- ✅ **Middleware/hooks support** - Extensible request/response pipeline
- ✅ **Type-safe error handling** - Custom error types for better debugging
- ✅ **Response helper methods** - Convenient JSON parsing and status checking
- ✅ **Minimal dependencies** - Only requires `coder/websocket`
- ✅ **Production-ready** - Secure defaults, proper logging, extensive godoc

## Installation

```bash
go get github.com/gurizzu/go-reqws
```

## Quick Start

### Basic HTTP Request

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/gurizzu/go-reqws"
)

func main() {
    client := reqws.NewClient("https://api.example.com", 30*time.Second)

    // Simple GET request - clean and concise!
    body, err := client.Request(context.Background(),
        reqws.GET("/users/123"),
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: %s", body)
}
```

### HTTP Request with Response Details

```go
resp, err := client.Do(context.Background(),
    reqws.POST("/users"),
    reqws.WithJSON(map[string]string{
        "name": "John Doe",
        "email": "john@example.com",
    }),
    reqws.WithBearerToken("YOUR_TOKEN"),
)

if err != nil {
    log.Fatal(err)
}

if !resp.IsSuccess() {
    log.Fatalf("Request failed: %d", resp.StatusCode)
}

var user User
if err := resp.JSON(&user); err != nil {
    log.Fatal(err)
}

log.Printf("Created user: %+v", user)
```

### WebSocket Streaming

```go
sendChan := make(chan interface{})
receiveChan := make(chan reqws.WebSocketResponse)

go client.WebSocketStream(context.Background(), sendChan, receiveChan,
    reqws.WithPath("/ws/stream"),
    reqws.WithQueryParam("token", "YOUR_TOKEN"),
)

// Send message
sendChan <- map[string]string{"action": "subscribe", "channel": "updates"}

// Receive messages
for msg := range receiveChan {
    if msg.Error != nil {
        log.Printf("Error: %v", msg.Error)
        break
    }
    if msg.Closed {
        log.Println("Connection closed")
        break
    }
    log.Printf("Received: %v", msg.Data)
}
```

## Advanced Usage

### Retry Mechanism

Automatically retry failed requests with exponential backoff:

```go
// Use default retry (3 attempts, 100ms initial delay, 5s max delay)
body, err := client.Request(ctx,
    reqws.GET("/api/data"),
    reqws.WithDefaultRetry(),
)

// Custom retry configuration
body, err := client.Request(ctx,
    reqws.GET("/api/data"),
    reqws.WithRetry(reqws.RetryConfig{
        MaxRetries:   5,
        InitialDelay: 200 * time.Millisecond,
        MaxDelay:     10 * time.Second,
        Multiplier:   2.0, // Exponential backoff
    }),
)
```

**Retry Logic:**
- ✅ Retries on: 5xx errors, 429 (rate limit), network errors
- ❌ No retry on: 4xx client errors (except 429)
- Exponential backoff: 100ms → 200ms → 400ms → 800ms → max 5s

### WebSocket Auto-Reconnection

Automatic reconnection when WebSocket connection drops:

```go
// Default reconnection (10 attempts, 1s initial delay, 30s max)
err := client.WebSocketStreamWithReconnect(ctx, sendChan, receiveChan,
    reqws.WithPath("/ws/stream"),
    reqws.WithDefaultWebSocketReconnect(),
)

// Custom reconnection with callback
reconnectCount := 0
err := client.WebSocketStreamWithReconnect(ctx, sendChan, receiveChan,
    reqws.WithPath("/ws/stream"),
    reqws.WithWebSocketAutoReconnect(reqws.WebSocketConfig{
        AutoReconnect:        true,
        MaxReconnectAttempts: 5,
        ReconnectDelay:       2 * time.Second,
        MaxReconnectDelay:    60 * time.Second,
        ReconnectMultiplier:  2.0,
        OnReconnect: func() {
            reconnectCount++
            log.Printf("Reconnecting... attempt #%d", reconnectCount)
        },
    }),
)
```

### Middleware/Hooks

Inject custom logic into the request/response pipeline:

#### Logging

```go
resp, err := client.Do(ctx,
    reqws.GET("/api/users"),
    reqws.WithBeforeRequest(func(req *http.Request) error {
        log.Printf("→ %s %s", req.Method, req.URL)
        return nil
    }),
    reqws.WithAfterResponse(func(req *http.Request, resp *http.Response) error {
        log.Printf("← %d %s", resp.StatusCode, req.URL)
        return nil
    }),
    reqws.WithOnError(func(req *http.Request, err error) {
        log.Printf("✘ Error for %s: %v", req.URL, err)
    }),
)
```

#### Metrics/Tracing

```go
startTime := time.Now()
resp, err := client.Do(ctx,
    reqws.GET("/api/users"),
    reqws.WithAfterResponse(func(req *http.Request, resp *http.Response) error {
        duration := time.Since(startTime)
        metrics.RecordHTTPRequest(req.Method, resp.StatusCode, duration)
        return nil
    }),
)
```

#### Dynamic Authentication

```go
resp, err := client.Do(ctx,
    reqws.GET("/api/protected"),
    reqws.WithBeforeRequest(func(req *http.Request) error {
        token, err := getAuthToken() // Your auth logic
        if err != nil {
            return err
        }
        req.Header.Set("Authorization", "Bearer "+token)
        return nil
    }),
)
```

### Custom Logger

Integrate with your existing logging solution (slog, zap, logrus, etc.):

```go
// Example with slog
type SlogAdapter struct {
    logger *slog.Logger
}

func (s SlogAdapter) Debug(msg string, keysAndValues ...interface{}) {
    s.logger.Debug(msg, keysAndValues...)
}

func (s SlogAdapter) Info(msg string, keysAndValues ...interface{}) {
    s.logger.Info(msg, keysAndValues...)
}

func (s SlogAdapter) Error(msg string, keysAndValues ...interface{}) {
    s.logger.Error(msg, keysAndValues...)
}

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    client := reqws.NewClient("https://api.example.com", 30*time.Second).
        WithLogger(SlogAdapter{logger})

    // All requests will now use your logger
    client.Request(ctx, reqws.GET("/users"))
}
```

### Error Handling

Type-safe error handling with custom error types:

```go
body, err := client.Request(ctx, reqws.GET("/api/users"))
if err != nil {
    // Check for HTTP errors
    var httpErr *reqws.HTTPError
    if errors.As(err, &httpErr) {
        log.Printf("HTTP Error: %d", httpErr.StatusCode)
        log.Printf("Response body: %s", httpErr.Body)

        if httpErr.StatusCode == 404 {
            // Handle not found
        } else if httpErr.StatusCode >= 500 {
            // Handle server error
        }
    }
    return err
}

// WebSocket errors
err := client.WebSocketStream(ctx, sendChan, receiveChan, ...)
if err != nil {
    var wsErr *reqws.WebSocketError
    if errors.As(err, &wsErr) {
        log.Printf("WebSocket Error: %s", wsErr.Reason)
        log.Printf("Underlying error: %v", wsErr.Err)
    }
}
```

### File Upload

Upload files with multipart form data:

```go
// Assuming you have a multipart.FileHeader from a form upload
resp, err := client.Do(ctx,
    reqws.POST("/upload"),
    reqws.WithFile("avatar", fileHeader),
    reqws.WithForm("user_id", "123"),
    reqws.WithForm("description", "Profile picture"),
)
```

## API Reference

### Client Creation

```go
// NewClient creates a new HTTP client
client := reqws.NewClient(baseURL string, timeout time.Duration) *Client

// WithLogger sets a custom logger
client.WithLogger(logger Logger) *Client
```

### HTTP Method Shortcuts

```go
// Combines method + path in one call
GET(path string) RequestOption
POST(path string) RequestOption
PUT(path string) RequestOption
DELETE(path string) RequestOption
PATCH(path string) RequestOption
HEAD(path string) RequestOption
OPTIONS(path string) RequestOption
```

### Request Options

```go
// HTTP method and path (legacy - use shortcuts above instead)
WithMethod(method string) RequestOption // For custom methods like PROPFIND
WithPath(path string) RequestOption

// Query parameters
WithQueryParam(key, value string) RequestOption
WithQueryParams(params url.Values) RequestOption

// Request body
WithJSON(body interface{}) RequestOption // Explicit JSON body (recommended)
WithBody(body interface{}) RequestOption // Alias for WithJSON

// Headers and authentication
WithHeader(key, value string) RequestOption
WithBearerToken(token string) RequestOption // Auto adds "Bearer " prefix
WithBasicAuth(username, password string) RequestOption // Auto base64 encodes
WithAuth(token string) RequestOption // Generic auth (full header value)

// Form data and file upload
WithForm(key, value string) RequestOption
WithFile(formFieldName string, file *multipart.FileHeader) RequestOption

// Retry configuration
WithRetry(config RetryConfig) RequestOption
WithDefaultRetry() RequestOption

// WebSocket configuration
WithWebSocketAutoReconnect(config WebSocketConfig) RequestOption
WithDefaultWebSocketReconnect() RequestOption

// Security
WithInsecureSkipVerify() RequestOption // ⚠️ Only for testing!

// Middleware/Hooks
WithBeforeRequest(hook RequestHook) RequestOption
WithAfterResponse(hook ResponseHook) RequestOption
WithOnError(hook ErrorHook) RequestOption
```

### Request Methods

```go
// Request executes HTTP request and returns body bytes
// Returns error for non-2xx status codes
Request(ctx context.Context, opts ...RequestOption) ([]byte, error)

// Do returns full Response object
// Does NOT return error for non-2xx status codes (manual checking required)
Do(ctx context.Context, opts ...RequestOption) (*Response, error)

// WebSocketStream establishes WebSocket connection
WebSocketStream(ctx context.Context, sendChan <-chan interface{}, receiveChan chan<- WebSocketResponse, opts ...RequestOption) error

// WebSocketStreamWithReconnect with automatic reconnection
WebSocketStreamWithReconnect(ctx context.Context, sendChan <-chan interface{}, receiveChan chan<- WebSocketResponse, opts ...RequestOption) error
```

### Response Methods

```go
// JSON unmarshals response body to struct
resp.JSON(v interface{}) error

// String returns response body as string
resp.String() string

// Status code helpers
resp.IsSuccess() bool       // 2xx
resp.IsClientError() bool   // 4xx
resp.IsServerError() bool   // 5xx
```

## Configuration Types

### RetryConfig

```go
type RetryConfig struct {
    MaxRetries   int           // Maximum retry attempts (default: 3)
    InitialDelay time.Duration // Initial delay (default: 100ms)
    MaxDelay     time.Duration // Maximum delay (default: 5s)
    Multiplier   float64       // Backoff multiplier (default: 2.0)
}
```

### WebSocketConfig

```go
type WebSocketConfig struct {
    AutoReconnect        bool          // Enable auto-reconnection
    MaxReconnectAttempts int           // Max reconnection attempts (0 = infinite)
    ReconnectDelay       time.Duration // Initial reconnection delay (default: 1s)
    MaxReconnectDelay    time.Duration // Maximum reconnection delay (default: 30s)
    ReconnectMultiplier  float64       // Backoff multiplier (default: 2.0)
    OnReconnect          func()        // Callback on each reconnection attempt
}
```

## Comparison with Other Libraries

| Feature | go-reqws | imroc/req | net/http |
|---------|----------|-----------|----------|
| HTTP Requests | ✅ | ✅ | ✅ |
| WebSocket Support | ✅ Built-in | ❌ | Manual setup |
| Auto Retry | ✅ Smart logic | ✅ | Manual |
| WS Auto-Reconnect | ✅ Built-in | ❌ | Manual |
| Middleware/Hooks | ✅ | ✅ | Manual |
| Response Helpers | ✅ | ✅ | Manual |
| Complexity | ⭐⭐⭐⭐⭐ Simple | ⭐⭐⭐ Moderate | ⭐⭐ Low-level |
| Use Case | Microservices + Real-time | Comprehensive HTTP | Full control |

**When to use go-reqws:**
- You need both HTTP and WebSocket in one library
- Building microservices with real-time features
- Want simplicity without sacrificing features
- Need auto-reconnection for WebSocket

**When NOT to use go-reqws:**
- You only need HTTP (use `imroc/req` or standard `net/http`)
- You need advanced HTTP features (HTTP/2 push, etc.)
- Maximum performance is critical (use lower-level libraries)

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/gurizzu/go-reqws"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Create client with custom logger
    client := reqws.NewClient("https://api.example.com", 30*time.Second)

    // HTTP Request with all features
    resp, err := client.Do(context.Background(),
        // Request configuration - clean and concise!
        reqws.GET("/api/users"),
        reqws.WithQueryParam("status", "active"),
        reqws.WithBearerToken("YOUR_TOKEN"),

        // Retry on failure
        reqws.WithDefaultRetry(),

        // Hooks for logging and metrics
        reqws.WithBeforeRequest(func(req *http.Request) error {
            log.Printf("→ Sending: %s %s", req.Method, req.URL)
            return nil
        }),
        reqws.WithAfterResponse(func(req *http.Request, resp *http.Response) error {
            log.Printf("← Received: %d", resp.StatusCode)
            return nil
        }),
        reqws.WithOnError(func(req *http.Request, err error) {
            log.Printf("✘ Error: %v", err)
        }),
    )

    if err != nil {
        var httpErr *reqws.HTTPError
        if errors.As(err, &httpErr) {
            log.Printf("HTTP Error: %d - %s", httpErr.StatusCode, httpErr.Body)
        }
        log.Fatal(err)
    }

    // Use response helpers
    if !resp.IsSuccess() {
        log.Fatalf("Request failed with status %d", resp.StatusCode)
    }

    var users []User
    if err := resp.JSON(&users); err != nil {
        log.Fatal(err)
    }

    log.Printf("Fetched %d users", len(users))

    // WebSocket with auto-reconnection
    sendChan := make(chan interface{})
    receiveChan := make(chan reqws.WebSocketResponse)

    go func() {
        err := client.WebSocketStreamWithReconnect(context.Background(),
            sendChan, receiveChan,
            reqws.WithPath("/ws/updates"),
            reqws.WithDefaultWebSocketReconnect(),
        )
        if err != nil {
            log.Printf("WebSocket error: %v", err)
        }
    }()

    // Send subscription message
    sendChan <- map[string]string{
        "action":  "subscribe",
        "channel": "user-updates",
    }

    // Receive real-time updates
    for msg := range receiveChan {
        if msg.Error != nil {
            log.Printf("WebSocket error: %v", msg.Error)
            continue
        }
        if msg.Closed {
            log.Println("Connection closed")
            break
        }
        log.Printf("Update received: %v", msg.Data)
    }
}
```

## Security Considerations

### TLS Certificate Verification

By default, `go-reqws` uses **secure TLS certificate verification**. Only disable it for testing/development:

```go
// ⚠️ INSECURE - Only use for testing!
client.WebSocketStream(ctx, sendChan, receiveChan,
    reqws.WithPath("/ws/stream"),
    reqws.WithInsecureSkipVerify(), // Disables TLS verification
)
```

**Never use `WithInsecureSkipVerify()` in production!** This makes your application vulnerable to man-in-the-middle attacks.

### Logging Sensitive Data

Be careful when using hooks to avoid logging sensitive information:

```go
// ❌ BAD - Might log sensitive headers/body
reqws.WithBeforeRequest(func(req *http.Request) error {
    log.Printf("Request: %+v", req) // Could leak auth tokens!
    return nil
})

// ✅ GOOD - Log only non-sensitive data
reqws.WithBeforeRequest(func(req *http.Request) error {
    log.Printf("Request: %s %s", req.Method, req.URL.Path)
    return nil
})
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

### Development Setup

```bash
# Clone repository
git clone https://github.com/gurizzu/go-reqws.git
cd go-reqws

# Run tests
go test -v ./...

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Lint
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by:
- [imroc/req](https://github.com/imroc/req) - Comprehensive Go HTTP client
- [coder/websocket](https://github.com/coder/websocket) - Excellent WebSocket library
- Go community for feedback and best practices

## Support

- **Issues:** [GitHub Issues](https://github.com/gurizzu/go-reqws/issues)
- **Discussions:** [GitHub Discussions](https://github.com/gurizzu/go-reqws/discussions)

---

**Made with ❤️ for the Go community**
