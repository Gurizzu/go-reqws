package reqws

import (
	"context"
	"net/http"
	"time"
)

// RetryConfig defines the configuration for retry behavior.
type RetryConfig struct {
	MaxRetries   int           // Maximum number of retry attempts (default: 3)
	InitialDelay time.Duration // Initial delay before first retry (default: 100ms)
	MaxDelay     time.Duration // Maximum delay between retries (default: 5s)
	Multiplier   float64       // Backoff multiplier (default: 2.0)
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// WithRetry enables retry with custom configuration.
func WithRetry(config RetryConfig) RequestOption {
	return func(c *requestConfig) {
		c.retryConfig = &config
	}
}

// WithDefaultRetry enables retry with default configuration.
// - MaxRetries: 3
// - InitialDelay: 100ms
// - MaxDelay: 5s
// - Multiplier: 2.0 (exponential backoff)
func WithDefaultRetry() RequestOption {
	config := DefaultRetryConfig()
	return func(c *requestConfig) {
		c.retryConfig = &config
	}
}

// shouldRetry determines if a request should be retried based on the response.
// Returns true for:
// - Network errors (no response)
// - 5xx server errors
// - 429 Too Many Requests
// Returns false for:
// - 2xx success
// - 4xx client errors (except 429)
func shouldRetry(resp *http.Response, err error) bool {
	// Network error, should retry
	if err != nil {
		return true
	}

	// No response, should retry
	if resp == nil {
		return true
	}

	// Retry on server errors (5xx)
	if resp.StatusCode >= 500 {
		return true
	}

	// Retry on rate limit (429)
	if resp.StatusCode == 429 {
		return true
	}

	// Don't retry on client errors (4xx except 429)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return false
	}

	// Success, don't retry
	return false
}

// executeWithRetry wraps the request execution with retry logic.
func (c *Client) executeWithRetry(ctx context.Context, config *requestConfig) (*http.Response, error) {
	// No retry config, execute once
	if config.retryConfig == nil {
		return c.buildAndExecuteRequest(ctx, config)
	}

	var lastResp *http.Response
	var lastErr error
	delay := config.retryConfig.InitialDelay

	for attempt := 0; attempt <= config.retryConfig.MaxRetries; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Execute request
		resp, err := c.buildAndExecuteRequest(ctx, config)

		// Success - return immediately
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Check if we should retry
		if !shouldRetry(resp, err) {
			// Don't retry, return error immediately
			return resp, err
		}

		// Store last response/error
		lastResp = resp
		lastErr = err

		// Close response body if exists (to avoid leaking connections)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		// Last attempt, don't sleep
		if attempt >= config.retryConfig.MaxRetries {
			break
		}

		// Log retry attempt if logger available
		if c.logger != nil {
			c.logger.Info("retrying request",
				"attempt", attempt+1,
				"max_retries", config.retryConfig.MaxRetries,
				"delay", delay,
			)
		}

		// Sleep with exponential backoff
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.retryConfig.Multiplier)
			if delay > config.retryConfig.MaxDelay {
				delay = config.retryConfig.MaxDelay
			}
		}
	}

	// All retries exhausted
	if lastErr != nil {
		if c.logger != nil {
			c.logger.Error("max retries exceeded", "error", lastErr)
		}
		return lastResp, lastErr
	}

	// Return last response if no error (non-2xx status)
	return lastResp, nil
}
