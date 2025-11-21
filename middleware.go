package reqws

import "net/http"

// RequestHook is a function that runs before a request is sent.
// It receives the prepared http.Request and can modify it or return an error to abort the request.
type RequestHook func(req *http.Request) error

// ResponseHook is a function that runs after a response is received.
// It receives both the original request and the response.
// Return an error to treat the response as failed.
type ResponseHook func(req *http.Request, resp *http.Response) error

// ErrorHook is a function that runs when an error occurs during the request.
// It receives the original request and the error that occurred.
// This hook cannot modify the error, it's primarily for logging/monitoring.
type ErrorHook func(req *http.Request, err error)

// WithBeforeRequest adds a hook that runs before the HTTP request is sent.
// Multiple hooks can be added and will be executed in the order they were added.
// If any hook returns an error, the request is aborted.
//
// Use cases:
// - Add custom headers
// - Log request details
// - Modify request body
// - Add authentication tokens dynamically
func WithBeforeRequest(hook RequestHook) RequestOption {
	return func(c *requestConfig) {
		c.beforeRequestHooks = append(c.beforeRequestHooks, hook)
	}
}

// WithAfterResponse adds a hook that runs after receiving the HTTP response.
// Multiple hooks can be added and will be executed in the order they were added.
// If any hook returns an error, the response is treated as failed.
//
// Use cases:
// - Log response details
// - Record metrics (latency, status codes)
// - Validate response structure
// - Custom retry logic based on response
func WithAfterResponse(hook ResponseHook) RequestOption {
	return func(c *requestConfig) {
		c.afterResponseHooks = append(c.afterResponseHooks, hook)
	}
}

// WithOnError adds a hook that runs when an error occurs.
// Multiple hooks can be added and will be executed in the order they were added.
// These hooks cannot modify the error, they're for observability.
//
// Use cases:
// - Log errors
// - Send error alerts
// - Record error metrics
// - Trace error propagation
func WithOnError(hook ErrorHook) RequestOption {
	return func(c *requestConfig) {
		c.errorHooks = append(c.errorHooks, hook)
	}
}
