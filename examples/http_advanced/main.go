package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gurizzu/go-reqws"
)

func main() {
	// Create client with custom timeout
	client := reqws.NewClient("https://jsonplaceholder.typicode.com", 30*time.Second)

	// Example 1: Request with retry
	log.Println("Example 1: Request with automatic retry")
	resp, err := client.Do(context.Background(),
		reqws.GET("/posts/1"),
		reqws.WithDefaultRetry(), // Enable retry with default config
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("✓ Success! Status: %d", resp.StatusCode)
	}

	// Example 2: Custom retry configuration
	log.Println("\nExample 2: Custom retry configuration")
	_, err = client.Request(context.Background(),
		reqws.GET("/posts/1"),
		reqws.WithRetry(reqws.RetryConfig{
			MaxRetries:   5,
			InitialDelay: 200 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
		}),
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("✓ Request completed with custom retry")
	}

	// Example 3: Request with middleware hooks
	log.Println("\nExample 3: Request with middleware/hooks")
	startTime := time.Now()

	resp, err = client.Do(context.Background(),
		reqws.GET("/posts"),

		// Before request hook - add custom header
		reqws.WithBeforeRequest(func(req *http.Request) error {
			log.Printf("→ Sending: %s %s", req.Method, req.URL.Path)
			req.Header.Set("X-Custom-Header", "my-value")
			return nil
		}),

		// After response hook - log response time
		reqws.WithAfterResponse(func(req *http.Request, resp *http.Response) error {
			duration := time.Since(startTime)
			log.Printf("← Received: %d (took %v)", resp.StatusCode, duration)
			return nil
		}),

		// Error hook - log errors
		reqws.WithOnError(func(req *http.Request, err error) {
			log.Printf("✘ Error for %s: %v", req.URL.Path, err)
		}),
	)

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("✓ Request completed successfully")
	}

	// Example 4: Multiple hooks for logging and metrics
	log.Println("\nExample 4: Multiple hooks for different purposes")

	requestCount := 0
	totalDuration := time.Duration(0)

	for i := 1; i <= 3; i++ {
		requestStart := time.Now()

		resp, err := client.Do(context.Background(),
			reqws.GET("/posts/1"),

			// Hook 1: Count requests
			reqws.WithBeforeRequest(func(req *http.Request) error {
				requestCount++
				log.Printf("[Metrics] Request #%d", requestCount)
				return nil
			}),

			// Hook 2: Record duration
			reqws.WithAfterResponse(func(req *http.Request, resp *http.Response) error {
				duration := time.Since(requestStart)
				totalDuration += duration
				log.Printf("[Metrics] Duration: %v", duration)
				return nil
			}),
		)

		if err != nil {
			log.Printf("Error: %v", err)
		} else {
			log.Printf("Request %d: Status %d", i, resp.StatusCode)
		}

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("\n[Summary] Total requests: %d, Avg duration: %v",
		requestCount, totalDuration/time.Duration(requestCount))

	// Example 5: Advanced error handling
	log.Println("\nExample 5: Advanced error handling with type assertions")

	_, err = client.Request(context.Background(),
		reqws.GET("/posts/99999"), // Will return 404
	)

	if err != nil {
		// Check for specific error types
		var httpErr *reqws.HTTPError
		if errors.As(err, &httpErr) {
			log.Printf("HTTP Error detected!")
			log.Printf("  Status Code: %d", httpErr.StatusCode)
			log.Printf("  Response Body: %s", string(httpErr.Body))

			// Handle different status codes
			switch {
			case httpErr.StatusCode == 404:
				log.Printf("  → Resource not found")
			case httpErr.StatusCode >= 500:
				log.Printf("  → Server error, might want to retry")
			case httpErr.StatusCode >= 400:
				log.Printf("  → Client error, fix the request")
			}
		} else {
			log.Printf("Other error: %v", err)
		}
	}

	// Example 6: Context with timeout
	log.Println("\nExample 6: Request with context timeout")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err = client.Do(ctx,
		reqws.GET("/posts/1"),
		reqws.WithBeforeRequest(func(req *http.Request) error {
			log.Printf("Request will timeout after 5 seconds")
			return nil
		}),
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("✘ Request timed out!")
		} else {
			log.Printf("Error: %v", err)
		}
	} else {
		log.Printf("✓ Completed within timeout: %d", resp.StatusCode)
	}
}
