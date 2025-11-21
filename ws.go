package reqws

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type WebSocketResponse struct {
	Data    interface{}
	RawData []byte
	Error   error
	Closed  bool
}

// WebSocketConfig defines configuration for WebSocket connections.
type WebSocketConfig struct {
	AutoReconnect        bool          // Enable automatic reconnection on disconnect
	MaxReconnectAttempts int           // Maximum number of reconnection attempts (0 = infinite)
	ReconnectDelay       time.Duration // Initial delay before reconnection
	MaxReconnectDelay    time.Duration // Maximum delay between reconnections
	ReconnectMultiplier  float64       // Backoff multiplier for reconnection delay
	OnReconnect          func()        // Callback function called on each reconnection attempt
}

// DefaultWebSocketConfig returns a sensible default WebSocket configuration.
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		AutoReconnect:        true,
		MaxReconnectAttempts: 10,
		ReconnectDelay:       1 * time.Second,
		MaxReconnectDelay:    30 * time.Second,
		ReconnectMultiplier:  2.0,
		OnReconnect:          nil,
	}
}

// WithWebSocketAutoReconnect enables WebSocket auto-reconnection with custom configuration.
func WithWebSocketAutoReconnect(config WebSocketConfig) RequestOption {
	return func(c *requestConfig) {
		c.wsConfig = &config
	}
}

// WithDefaultWebSocketReconnect enables WebSocket auto-reconnection with default configuration.
// - MaxReconnectAttempts: 10
// - ReconnectDelay: 1s
// - MaxReconnectDelay: 30s
// - ReconnectMultiplier: 2.0 (exponential backoff)
func WithDefaultWebSocketReconnect() RequestOption {
	config := DefaultWebSocketConfig()
	return func(c *requestConfig) {
		c.wsConfig = &config
	}
}

// WebSocketStream - Persistent connection with channel-based communication
func (c *Client) WebSocketStream(ctx context.Context, sendChan <-chan interface{}, receiveChan chan<- WebSocketResponse, opts ...RequestOption) error {
	config := &requestConfig{
		queryParams: url.Values{},
		headers:     http.Header{},
	}

	for _, opt := range opts {
		opt(config)
	}

	fullURL, err := url.Parse(c.baseURL + config.path)
	if err != nil {
		return err
	}
	fullURL.RawQuery = config.queryParams.Encode()

	if c.logger != nil {
		c.logger.Info("opening WebSocket stream", "url", fullURL.String())
	}

	// Default DialOptions
	dialOpts := &websocket.DialOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	}

	// Only skip TLS verification if explicitly requested via WithInsecureSkipVerify()
	// Default: Secure TLS verification (InsecureSkipVerify = false)
	if config.insecureSkipVerify && (strings.HasPrefix(fullURL.String(), "https://") || strings.HasPrefix(fullURL.String(), "wss://")) {
		dialOpts.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	conn, resp, err := websocket.Dial(ctx, fullURL.String(), dialOpts)
	if err != nil {
		if resp != nil {
			return NewWebSocketError(fmt.Sprintf("dial failed with status %d", resp.StatusCode), err)
		}
		return NewWebSocketError("dial failed", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "closing stream")

	conn.SetReadLimit(1024 * 1024) // 1MB

	// Goroutine for reading messages
	go func() {
		defer close(receiveChan)
		for {
			var msg map[string]interface{}
			err := wsjson.Read(ctx, conn, &msg)
			if err != nil {
				receiveChan <- WebSocketResponse{
					Error:  err,
					Closed: true,
				}
				return
			}
			receiveChan <- WebSocketResponse{
				Data:   msg,
				Closed: false,
			}
		}
	}()

	// Goroutine for writing messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-sendChan:
			if !ok {
				// Send channel closed, close connection
				return nil
			}
			err := wsjson.Write(ctx, conn, msg)
			if err != nil {
				return NewWebSocketError("failed to send message", err)
			}
			if c.logger != nil {
				c.logger.Debug("message sent to WebSocket stream")
			}
		}
	}
}

// WebSocketStreamWithReconnect wraps WebSocketStream with automatic reconnection logic.
// If the connection drops, it will automatically attempt to reconnect with exponential backoff.
// Use WithWebSocketAutoReconnect() or WithDefaultWebSocketReconnect() to configure reconnection behavior.
func (c *Client) WebSocketStreamWithReconnect(ctx context.Context, sendChan <-chan interface{}, receiveChan chan<- WebSocketResponse, opts ...RequestOption) error {
	// Parse config from options
	config := &requestConfig{
		queryParams: url.Values{},
		headers:     http.Header{},
	}
	for _, opt := range opts {
		opt(config)
	}

	// If no WebSocket config or auto-reconnect disabled, just call normal WebSocketStream
	if config.wsConfig == nil || !config.wsConfig.AutoReconnect {
		return c.WebSocketStream(ctx, sendChan, receiveChan, opts...)
	}

	// Auto-reconnect enabled
	attempt := 0
	delay := config.wsConfig.ReconnectDelay

	for {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Log reconnection attempt if not first attempt
		if attempt > 0 {
			if c.logger != nil {
				c.logger.Info("attempting to reconnect WebSocket",
					"attempt", attempt,
					"max_attempts", config.wsConfig.MaxReconnectAttempts,
					"delay", delay,
				)
			}

			// Call OnReconnect callback if provided
			if config.wsConfig.OnReconnect != nil {
				config.wsConfig.OnReconnect()
			}

			// Sleep with exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Calculate next delay
				delay = time.Duration(float64(delay) * config.wsConfig.ReconnectMultiplier)
				if delay > config.wsConfig.MaxReconnectDelay {
					delay = config.wsConfig.MaxReconnectDelay
				}
			}
		}

		// Attempt connection
		err := c.WebSocketStream(ctx, sendChan, receiveChan, opts...)

		// If context was cancelled, don't reconnect
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check if we should stop reconnecting
		attempt++
		if config.wsConfig.MaxReconnectAttempts > 0 && attempt >= config.wsConfig.MaxReconnectAttempts {
			if c.logger != nil {
				c.logger.Error("max WebSocket reconnection attempts reached",
					"attempts", attempt,
					"error", err,
				)
			}
			return NewWebSocketError("max reconnection attempts exceeded", err)
		}

		// Log disconnection
		if c.logger != nil {
			c.logger.Info("WebSocket disconnected, will retry", "error", err)
		}

		// Continue to next iteration for reconnection
	}
}
