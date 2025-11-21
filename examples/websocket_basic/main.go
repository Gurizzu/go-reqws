package main

import (
	"context"
	"log"
	"time"

	"github.com/gurizzu/go-reqws"
)

func main() {
	log.Println("WebSocket Basic Example")
	log.Println("Connecting to WebSocket echo server...")

	// Create client
	client := reqws.NewClient("wss://echo.websocket.org", 30*time.Second)

	// Create channels for sending and receiving
	sendChan := make(chan interface{})
	receiveChan := make(chan reqws.WebSocketResponse)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start WebSocket connection in goroutine
	go func() {
		err := client.WebSocketStream(ctx, sendChan, receiveChan,
			reqws.WithPath("/"),
		)
		if err != nil {
			log.Printf("WebSocket connection error: %v", err)
		}
	}()

	// Wait a bit for connection to establish
	time.Sleep(1 * time.Second)

	// Example 1: Send and receive text messages
	log.Println("\nExample 1: Sending text messages")

	messages := []string{
		"Hello, WebSocket!",
		"This is a test message",
		"Go is awesome!",
	}

	for _, msg := range messages {
		// Send message
		log.Printf("→ Sending: %s", msg)
		sendChan <- map[string]string{
			"type":    "message",
			"content": msg,
		}

		// Receive echo response
		select {
		case response := <-receiveChan:
			if response.Error != nil {
				log.Printf("✘ Error: %v", response.Error)
			} else if response.Closed {
				log.Printf("Connection closed")
				return
			} else {
				log.Printf("← Received: %v", response.Data)
			}
		case <-time.After(5 * time.Second):
			log.Printf("✘ Timeout waiting for response")
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Example 2: Send structured data
	log.Println("\nExample 2: Sending structured data")

	structuredMsg := map[string]interface{}{
		"action": "echo",
		"data": map[string]interface{}{
			"user_id":   123,
			"message":   "Structured message",
			"timestamp": time.Now().Unix(),
		},
	}

	log.Printf("→ Sending: %+v", structuredMsg)
	sendChan <- structuredMsg

	select {
	case response := <-receiveChan:
		if response.Error != nil {
			log.Printf("✘ Error: %v", response.Error)
		} else {
			log.Printf("← Received: %v", response.Data)
		}
	case <-time.After(5 * time.Second):
		log.Printf("✘ Timeout waiting for response")
	}

	// Example 3: Handle multiple responses
	log.Println("\nExample 3: Handling multiple responses")

	// Send multiple messages rapidly
	for i := 1; i <= 5; i++ {
		sendChan <- map[string]interface{}{
			"id":      i,
			"message": "Rapid message",
		}
		log.Printf("→ Sent message #%d", i)
	}

	// Receive all responses
	received := 0
	timeout := time.After(10 * time.Second)

receiveLoop:
	for received < 5 {
		select {
		case response := <-receiveChan:
			if response.Error != nil {
				log.Printf("✘ Error: %v", response.Error)
				break receiveLoop
			}
			if response.Closed {
				log.Printf("Connection closed")
				break receiveLoop
			}
			received++
			log.Printf("← Received #%d: %v", received, response.Data)

		case <-timeout:
			log.Printf("✘ Timeout after receiving %d messages", received)
			break receiveLoop
		}
	}

	log.Printf("\n✓ Received %d/%d messages", received, 5)

	// Close connection gracefully
	log.Println("\nClosing connection...")
	close(sendChan)
	time.Sleep(1 * time.Second)

	log.Println("✓ WebSocket example completed")
}
