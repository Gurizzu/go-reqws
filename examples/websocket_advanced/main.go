package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/gurizzu/go-reqws"
)

func main() {
	log.Println("WebSocket Advanced Example - Auto-Reconnection")

	// Create client
	client := reqws.NewClient("wss://echo.websocket.org", 30*time.Second)

	// Create channels
	sendChan := make(chan interface{})
	receiveChan := make(chan reqws.WebSocketResponse)

	// Track statistics
	var reconnectCount atomic.Int32
	var messagesSent atomic.Int32
	var messagesReceived atomic.Int32

	// Example 1: WebSocket with auto-reconnection
	log.Println("\nExample 1: WebSocket with auto-reconnection")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start WebSocket with auto-reconnection
	go func() {
		err := client.WebSocketStreamWithReconnect(ctx, sendChan, receiveChan,
			reqws.WithPath("/"),
			reqws.WithDefaultWebSocketReconnect(), // Auto-reconnect enabled
		)
		if err != nil {
			log.Printf("WebSocket error: %v", err)
		}
	}()

	// Wait for initial connection
	time.Sleep(2 * time.Second)
	log.Println("✓ Connected to WebSocket server")

	// Send messages continuously
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		messageID := 1
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msg := map[string]interface{}{
					"id":        messageID,
					"message":   "Periodic message",
					"timestamp": time.Now().Unix(),
				}

				select {
				case sendChan <- msg:
					messagesSent.Add(1)
					log.Printf("→ Sent message #%d", messageID)
					messageID++
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Receive messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case response := <-receiveChan:
				if response.Error != nil {
					log.Printf("✘ Receive error: %v", response.Error)
					continue
				}
				if response.Closed {
					log.Println("Connection closed, reconnecting...")
					continue
				}

				messagesReceived.Add(1)
				log.Printf("← Received message #%d: %v",
					messagesReceived.Load(), response.Data)
			}
		}
	}()

	// Example 2: Custom reconnection with callback
	log.Println("\nExample 2: Custom reconnection configuration")

	customCtx, customCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer customCancel()

	customSendChan := make(chan interface{})
	customReceiveChan := make(chan reqws.WebSocketResponse)

	go func() {
		err := client.WebSocketStreamWithReconnect(customCtx,
			customSendChan, customReceiveChan,
			reqws.WithPath("/"),
			reqws.WithWebSocketAutoReconnect(reqws.WebSocketConfig{
				AutoReconnect:        true,
				MaxReconnectAttempts: 5,
				ReconnectDelay:       2 * time.Second,
				MaxReconnectDelay:    30 * time.Second,
				ReconnectMultiplier:  2.0,
				OnReconnect: func() {
					count := reconnectCount.Add(1)
					log.Printf("🔄 Reconnecting... attempt #%d", count)
					log.Printf("   Delay: %v (exponential backoff)",
						time.Duration(float64(2*time.Second)*
							float64(count)*2.0))
				},
			}),
		)
		if err != nil {
			log.Printf("Custom WebSocket error: %v", err)
		}
	}()

	// Run for 20 seconds
	time.Sleep(20 * time.Second)

	// Print statistics
	log.Println("\n--- Statistics ---")
	log.Printf("Messages sent: %d", messagesSent.Load())
	log.Printf("Messages received: %d", messagesReceived.Load())
	log.Printf("Reconnection attempts: %d", reconnectCount.Load())

	// Example 3: Graceful shutdown
	log.Println("\nExample 3: Graceful shutdown")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	shutdownSendChan := make(chan interface{})
	shutdownReceiveChan := make(chan reqws.WebSocketResponse)

	// Start connection
	connectionDone := make(chan struct{})
	go func() {
		defer close(connectionDone)
		err := client.WebSocketStreamWithReconnect(shutdownCtx,
			shutdownSendChan, shutdownReceiveChan,
			reqws.WithPath("/"),
			reqws.WithDefaultWebSocketReconnect(),
		)
		if err != nil {
			log.Printf("Connection ended: %v", err)
		}
	}()

	// Send a few messages
	time.Sleep(1 * time.Second)
	for i := 1; i <= 3; i++ {
		shutdownSendChan <- map[string]string{
			"message": "Shutdown test",
		}
		log.Printf("→ Sent shutdown test message #%d", i)
		time.Sleep(500 * time.Millisecond)
	}

	// Graceful shutdown
	log.Println("\nInitiating graceful shutdown...")

	// Close send channel to signal no more messages
	close(shutdownSendChan)

	// Wait for connection to close or timeout
	select {
	case <-connectionDone:
		log.Println("✓ Connection closed gracefully")
	case <-time.After(5 * time.Second):
		log.Println("✘ Timeout waiting for connection to close")
		shutdownCancel() // Force cancel
	}

	log.Println("\n✓ WebSocket advanced example completed")
	log.Printf("Final stats - Sent: %d, Received: %d, Reconnects: %d",
		messagesSent.Load(), messagesReceived.Load(), reconnectCount.Load())
}
