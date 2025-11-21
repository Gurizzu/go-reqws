package reqws

import "fmt"

// HTTPError represents an HTTP error response with a non-2xx status code.
type HTTPError struct {
	StatusCode int
	Body       []byte
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("HTTP %d: received non-2xx status code", e.StatusCode)
}

// NewHTTPError creates a new HTTPError with the given status code and response body.
func NewHTTPError(statusCode int, body []byte) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Body:       body,
		Message:    fmt.Sprintf("received non-2xx status code: %d", statusCode),
	}
}

// WebSocketError represents a WebSocket-specific error.
type WebSocketError struct {
	Reason string
	Err    error
}

func (e *WebSocketError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("websocket error: %s: %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("websocket error: %s", e.Reason)
}

// Unwrap returns the underlying error for error chain support.
func (e *WebSocketError) Unwrap() error {
	return e.Err
}

// NewWebSocketError creates a new WebSocketError with the given reason and underlying error.
func NewWebSocketError(reason string, err error) *WebSocketError {
	return &WebSocketError{
		Reason: reason,
		Err:    err,
	}
}
