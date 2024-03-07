// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.

package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

const BUFFER_SIZE = 1024

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// FIXME: Implement some way to verify who's connecting
		return true // Allow connections from any origin
	},
}

func netConnHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		queryValues := r.URL.Query()
		network := queryValues.Get("network")
		address := queryValues.Get("address")

		// Check if network or address parameters are missing and return HTTP 400 if they are
		if network == "" || address == "" {
			http.Error(w, "Missing 'network' or 'address' query parameters", http.StatusBadRequest)
			return
		}

		log.Printf("New websocket connection to %s %s", network, address)

		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancel is called when this function exits

		// Upgrade the HTTP server connection to a WebSocket connection
		wsConn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
		}
		defer wsConn.Close()

		// Connect to remote target
		netConn, err := net.Dial(network, address)
		if err != nil {
			log.Printf("Error connecting %s address %s: %v", network, address, err)
			return
		}
		defer netConn.Close()

		// Start goroutine to read from wasm side and write to remote target
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					// Context was cancelled, exit the goroutine
					return
				default:
					messageType, message, err := wsConn.ReadMessage()
					if err != nil {
						log.Printf("Error reading from WebSocket: %v", err)
						return
					}
					// Assuming we only want to forward text/binary messages
					if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
						totalWritten := 0
						messageLength := len(message)
						for totalWritten < messageLength {
							n, err := netConn.Write(message[totalWritten:])
							if err != nil {
								log.Printf("Error writing to netConn: %v", err)
								return
							}
							totalWritten += n
						}
					}
				}
			}
		}(ctx)

		// Read from remote target and write to wasm side
		buffer := make([]byte, BUFFER_SIZE) // Adjust buffer size based on your needs
		for {
			select {

			case <-ctx.Done():
				// Context was cancelled, exit the main loop
				return

			default:
				n, err := netConn.Read(buffer)
				if err != nil {
					log.Printf("Error reading from netConn: %v", err)
					return
				}
				err = wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err != nil {
					log.Printf("Error writing to WebSocket: %v", err)
					return
				}
			}
		}

	}
}
