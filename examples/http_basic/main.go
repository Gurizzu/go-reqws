package main

import (
	"context"
	"log"
	"time"

	"github.com/gurizzu/go-reqws"
)

func main() {
	// Create client
	client := reqws.NewClient("https://jsonplaceholder.typicode.com", 30*time.Second)

	// Example 1: Simple GET request
	log.Println("Example 1: Simple GET request")
	body, err := client.Request(context.Background(),
		reqws.GET("/posts/1"),
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Response: %s", body)
	}

	// Example 2: GET with query parameters
	log.Println("\nExample 2: GET with query parameters")
	body, err = client.Request(context.Background(),
		reqws.GET("/posts"),
		reqws.WithQueryParam("userId", "1"),
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Response (first 200 chars): %s...", body[:200])
	}

	// Example 3: POST request with JSON body
	log.Println("\nExample 3: POST request with JSON body")
	resp, err := client.Do(context.Background(),
		reqws.POST("/posts"),
		reqws.WithJSON(map[string]interface{}{
			"title":  "My New Post",
			"body":   "This is the content of my post",
			"userId": 1,
		}),
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Status: %d", resp.StatusCode)
		log.Printf("Response: %s", resp.String())
	}

	// Example 4: Using response helpers
	log.Println("\nExample 4: Using response helpers")
	resp, err = client.Do(context.Background(),
		reqws.GET("/posts/1"),
	)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		if resp.IsSuccess() {
			log.Printf("✓ Request successful!")

			var post map[string]interface{}
			if err := resp.JSON(&post); err != nil {
				log.Printf("JSON parse error: %v", err)
			} else {
				log.Printf("Post title: %v", post["title"])
				log.Printf("Post body: %v", post["body"])
			}
		}
	}

	// Example 5: Error handling
	log.Println("\nExample 5: Error handling")
	_, err = client.Request(context.Background(),
		reqws.GET("/posts/999999"), // Non-existent resource
	)
	if err != nil {
		log.Printf("Expected error: %v", err)

		if httpErr, ok := err.(*reqws.HTTPError); ok {
			log.Printf("HTTP Error - Status: %d", httpErr.StatusCode)
			log.Printf("Response body: %s", httpErr.Body)
		}
	}
}
