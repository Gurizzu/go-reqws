package reqws

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Logger is an interface for logging operations.
// Users can provide their own implementation (slog, zap, logrus, etc.)
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// Client represents an HTTP/WebSocket client for making requests.
type Client struct {
	client  *http.Client
	baseURL string
	logger  Logger
}

// Requests is deprecated. Use Client instead.
// Kept for backward compatibility.
type Requests = Client

type requestConfig struct {
	method              string
	path                string
	queryParams         url.Values
	body                interface{}
	headers             http.Header
	auth                string
	file                *multipart.FileHeader
	formFieldName       string
	formFields          map[string]string
	insecureSkipVerify  bool
	retryConfig         *RetryConfig
	wsConfig            *WebSocketConfig
	beforeRequestHooks  []RequestHook
	afterResponseHooks  []ResponseHook
	errorHooks          []ErrorHook
}

type RequestOption func(*requestConfig)

// NewClient creates a new HTTP client with the specified base URL and timeout.
//
// The baseURL should not include a trailing slash. All request paths will be
// appended to this base URL.
//
// Example:
//
//	client := reqws.NewClient("https://api.example.com", 30*time.Second)
//	body, err := client.Request(ctx, reqws.GET("/users"))
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewRequests is deprecated. Use NewClient instead.
// Kept for backward compatibility.
func NewRequests(baseURL string, timeout time.Duration) *Client {
	return NewClient(baseURL, timeout)
}

// buildAndExecuteRequest is a helper method that builds and executes an HTTP request.
// It returns the raw http.Response which can be processed by the caller.
func (c *Client) buildAndExecuteRequest(ctx context.Context, config *requestConfig) (*http.Response, error) {
	// Build full URL with query parameters
	fullURL, err := url.Parse(c.baseURL + config.path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	fullURL.RawQuery = config.queryParams.Encode()

	var reqBody io.Reader
	var contentType string

	// Handle file upload with multipart form data
	if config.file != nil {
		bodyBuffer := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuffer)

		// Add form fields
		for k, v := range config.formFields {
			if err := writer.WriteField(k, v); err != nil {
				return nil, fmt.Errorf("failed to write form field: %w", err)
			}
		}

		// Add file
		sanitizedFilename := strings.ReplaceAll(config.file.Filename, " ", "_")
		part, err := writer.CreateFormFile(config.formFieldName, sanitizedFilename)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		file, err := config.file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy file to buffer: %w", err)
		}
		writer.Close()

		reqBody = bodyBuffer
		contentType = writer.FormDataContentType()
	} else if config.body != nil {
		// Handle JSON body
		jsonBody, err := json.Marshal(config.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		contentType = "application/json"
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, config.method, fullURL.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, values := range config.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if config.auth != "" {
		req.Header.Set("Authorization", config.auth)
	}

	// Execute before-request hooks
	for _, hook := range config.beforeRequestHooks {
		if err := hook(req); err != nil {
			// Call error hooks
			for _, errHook := range config.errorHooks {
				errHook(req, err)
			}
			return nil, fmt.Errorf("before-request hook failed: %w", err)
		}
	}

	// Log request if logger is available
	if c.logger != nil {
		c.logger.Debug("requesting to API", "method", config.method, "url", fullURL.String())
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		// Call error hooks
		for _, errHook := range config.errorHooks {
			errHook(req, err)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Execute after-response hooks
	for _, hook := range config.afterResponseHooks {
		if err := hook(req, resp); err != nil {
			// Call error hooks
			for _, errHook := range config.errorHooks {
				errHook(req, err)
			}
			resp.Body.Close()
			return nil, fmt.Errorf("after-response hook failed: %w", err)
		}
	}

	return resp, nil
}

// Request executes an HTTP request and returns only the response body as bytes.
// This is the simple method for most use cases - it automatically fails on non-2xx status codes.
//
// Returns an error if the status code is not 2xx.
// Supports retry via WithRetry() or WithDefaultRetry() options.
//
// Example:
//
//	body, err := client.Request(ctx,
//		reqws.GET("/users/1"),
//		reqws.WithBearerToken("token"),
//	)
func (c *Client) Request(ctx context.Context, opts ...RequestOption) ([]byte, error) {
	config := &requestConfig{
		method:      http.MethodGet,
		queryParams: url.Values{},
		headers:     http.Header{},
	}

	for _, opt := range opts {
		opt(config)
	}

	resp, err := c.executeWithRetry(ctx, config)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, NewHTTPError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// HTTP Method Shortcuts
// These functions combine method and path for cleaner code

// GET creates a GET request to the specified path.
// This is a shortcut for WithMethod("GET") + WithPath(path).
//
// Example:
//
//	body, err := client.Request(ctx, reqws.GET("/users"))
//	resp, err := client.Do(ctx, reqws.GET("/users"))
func GET(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "GET"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// POST creates a POST request to the specified path.
// This is a shortcut for WithMethod("POST") + WithPath(path).
//
// Example:
//
//	client.Do(ctx, reqws.POST("/users"), reqws.WithJSON(user))
func POST(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "POST"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// PUT creates a PUT request to the specified path.
// This is a shortcut for WithMethod("PUT") + WithPath(path).
//
// Example:
//
//	client.Do(ctx, reqws.PUT("/users/1"), reqws.WithJSON(user))
func PUT(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "PUT"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// DELETE creates a DELETE request to the specified path.
// This is a shortcut for WithMethod("DELETE") + WithPath(path).
//
// Example:
//
//	client.Request(ctx, reqws.DELETE("/users/1"))
func DELETE(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "DELETE"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// PATCH creates a PATCH request to the specified path.
// This is a shortcut for WithMethod("PATCH") + WithPath(path).
//
// Example:
//
//	client.Do(ctx, reqws.PATCH("/users/1"), reqws.WithJSON(updates))
func PATCH(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "PATCH"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// HEAD creates a HEAD request to the specified path.
// Useful for checking if a resource exists without downloading it.
//
// Example:
//
//	resp, err := client.Do(ctx, reqws.HEAD("/users/1"))
func HEAD(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "HEAD"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// OPTIONS creates an OPTIONS request to the specified path.
// Useful for CORS preflight requests.
//
// Example:
//
//	resp, err := client.Do(ctx, reqws.OPTIONS("/api"))
func OPTIONS(path string) RequestOption {
	return func(c *requestConfig) {
		c.method = "OPTIONS"
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// WithMethod sets a custom HTTP method for the request.
// For common methods (GET, POST, PUT, DELETE, PATCH), consider using the shortcut functions instead.
//
// Use this for custom or less common HTTP methods like PROPFIND, MKCOL, etc.
//
// Example:
//
//	client.Do(ctx, reqws.WithMethod("PROPFIND"), reqws.WithPath("/files"))
func WithMethod(method string) RequestOption {
	return func(c *requestConfig) {
		c.method = strings.ToUpper(method)
	}
}

// WithPath sets the request path.
// The path is automatically prefixed with "/" if not present.
//
// Note: For common cases, consider using HTTP method shortcuts (GET, POST, etc.)
// which combine method and path.
//
// Example:
//
//	// Legacy approach
//	client.Request(ctx, reqws.WithMethod("GET"), reqws.WithPath("/users"))
//
//	// Better: use shortcut instead
//	client.Request(ctx, reqws.GET("/users"))
func WithPath(path string) RequestOption {
	return func(c *requestConfig) {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		c.path = path
	}
}

// WithQueryParam adds a single query parameter to the request URL.
// Can be called multiple times to add multiple parameters.
//
// Example:
//
//	client.Request(ctx,
//		reqws.GET("/users"),
//		reqws.WithQueryParam("status", "active"),
//		reqws.WithQueryParam("limit", "10"),
//	)
func WithQueryParam(key, value string) RequestOption {
	return func(c *requestConfig) {
		c.queryParams.Add(key, value)
	}
}

// WithBody sets the request body.
// The body will be automatically marshaled to JSON.
//
// For more explicit JSON handling, consider using WithJSON() instead.
//
// Example:
//
//	client.Do(ctx, reqws.POST("/users"), reqws.WithBody(user))
func WithBody(body interface{}) RequestOption {
	return func(c *requestConfig) {
		c.body = body
	}
}

// WithJSON sets the request body as JSON.
// This is an explicit alias for WithBody() for better code clarity.
// The body will be marshaled to JSON automatically.
//
// Example:
//
//	client.Do(ctx,
//		reqws.POST("/users"),
//		reqws.WithJSON(map[string]string{
//			"name": "John Doe",
//			"email": "john@example.com",
//		}),
//	)
func WithJSON(body interface{}) RequestOption {
	return WithBody(body)
}

// WithHeader adds a custom HTTP header to the request.
// Can be called multiple times to add multiple headers.
//
// Example:
//
//	client.Request(ctx,
//		reqws.GET("/api/data"),
//		reqws.WithHeader("X-API-Version", "v1"),
//		reqws.WithHeader("X-Request-ID", "12345"),
//	)
func WithHeader(key, value string) RequestOption {
	return func(c *requestConfig) {
		c.headers.Add(key, value)
	}
}

// WithAuth sets the Authorization header with the provided token.
// The token should include the auth scheme (e.g., "Bearer xxx").
//
// For Bearer tokens specifically, consider using WithBearerToken() instead.
//
// Example:
//
//	client.Request(ctx, reqws.GET("/protected"), reqws.WithAuth("Bearer abc123"))
func WithAuth(token string) RequestOption {
	return func(c *requestConfig) {
		c.auth = token
	}
}

// WithBearerToken sets the Authorization header with a Bearer token.
// This is a convenience method that automatically prepends "Bearer " to the token.
//
// Example:
//
//	client.Request(ctx,
//		reqws.GET("/protected"),
//		reqws.WithBearerToken("abc123"),
//	)
func WithBearerToken(token string) RequestOption {
	return func(c *requestConfig) {
		c.auth = "Bearer " + token
	}
}

// WithBasicAuth sets the Authorization header with Basic authentication.
// The credentials will be automatically base64 encoded.
//
// Example:
//
//	client.Request(ctx,
//		reqws.GET("/protected"),
//		reqws.WithBasicAuth("username", "password"),
//	)
func WithBasicAuth(username, password string) RequestOption {
	return func(c *requestConfig) {
		// Standard Basic Auth encoding
		credentials := username + ":" + password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		c.auth = "Basic " + encoded
	}
}

// WithForm adds a form field for multipart/form-data requests.
// Use this together with WithFile() for file uploads.
//
// Example:
//
//	client.Do(ctx,
//		reqws.POST("/upload"),
//		reqws.WithFile("avatar", fileHeader),
//		reqws.WithForm("user_id", "123"),
//		reqws.WithForm("description", "Profile picture"),
//	)
func WithForm(key, value string) RequestOption {
	return func(c *requestConfig) {
		if c.formFields == nil {
			c.formFields = make(map[string]string)
		}
		c.formFields[key] = value
	}
}

// WithFile adds a file to the request for multipart/form-data upload.
// The formFieldName is the name of the form field (defaults to "file" if empty).
//
// Example:
//
//	client.Do(ctx,
//		reqws.POST("/upload"),
//		reqws.WithFile("avatar", fileHeader),
//	)
func WithFile(formFieldName string, file *multipart.FileHeader) RequestOption {
	return func(c *requestConfig) {
		c.file = file
		if formFieldName == "" {
			c.formFieldName = "file"
		} else {
			c.formFieldName = formFieldName
		}
	}
}

// WithQueryParams adds multiple query parameters at once from url.Values.
// For adding single parameters, use WithQueryParam() instead.
//
// Example:
//
//	params := url.Values{}
//	params.Add("status", "active")
//	params.Add("limit", "10")
//	client.Request(ctx, reqws.GET("/users"), reqws.WithQueryParams(params))
func WithQueryParams(params url.Values) RequestOption {
	return func(cfg *requestConfig) {
		if cfg.queryParams == nil {
			cfg.queryParams = url.Values{}
		}
		for key, values := range params {
			for _, v := range values {
				cfg.queryParams.Add(key, v)
			}
		}
	}
}

// WithInsecureSkipVerify disables TLS certificate verification.
// WARNING: This should only be used for testing or development.
// Using this in production makes your application vulnerable to man-in-the-middle attacks.
func WithInsecureSkipVerify() RequestOption {
	return func(c *requestConfig) {
		c.insecureSkipVerify = true
	}
}

// WithLogger sets a custom logger for the Client.
// The logger will be used for all HTTP and WebSocket operations.
// If no logger is provided, logging is disabled by default.
//
// Example:
//
//	client := reqws.NewClient("https://api.example.com", 30*time.Second).
//		WithLogger(myLogger)
func (c *Client) WithLogger(logger Logger) *Client {
	c.logger = logger
	return c
}

// Response represents an HTTP response with helper methods.
type Response struct {
	Body       []byte
	Headers    http.Header
	StatusCode int
}

// JSON unmarshals the response body into the provided value.
// The value should be a pointer to the target struct.
func (r *Response) JSON(v interface{}) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// String returns the response body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// IsSuccess returns true if the status code is 2xx (200-299).
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError returns true if the status code is 4xx (400-499).
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the status code is 5xx (500-599).
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// Do executes an HTTP request and returns the full Response object with body, headers, and status code.
// This method gives you full control - it does NOT automatically fail on non-2xx status codes.
//
// Use this when you need:
// - Access to response headers
// - Manual handling of different status codes
// - Response helper methods (JSON, IsSuccess, etc.)
//
// Unlike Request(), this does not return an error for non-2xx status codes.
// You must manually check resp.IsSuccess() or resp.StatusCode.
// Supports retry via WithRetry() or WithDefaultRetry() options.
//
// Example:
//
//	resp, err := client.Do(ctx,
//		reqws.GET("/users/1"),
//		reqws.WithBearerToken("token"),
//	)
//	if err != nil {
//		return err
//	}
//	if !resp.IsSuccess() {
//		return fmt.Errorf("failed: %d", resp.StatusCode)
//	}
//	var user User
//	resp.JSON(&user)
func (c *Client) Do(ctx context.Context, opts ...RequestOption) (*Response, error) {
	config := &requestConfig{
		method:      http.MethodGet,
		queryParams: url.Values{},
		headers:     http.Header{},
	}

	for _, opt := range opts {
		opt(config)
	}

	resp, err := c.executeWithRetry(ctx, config)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		Body:       respBody,
		Headers:    resp.Header.Clone(),
		StatusCode: resp.StatusCode,
	}, nil
}
