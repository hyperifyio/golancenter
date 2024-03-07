// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.

package main

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

// tcpProxyHandler handles incoming WebSocket connections, upgrading them and then
// forwarding the connection to the VNC server.
func tcpProxyHandler(w http.ResponseWriter, r *http.Request) {

	// Upgrade the HTTP server connection to a WebSocket connection
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer wsConn.Close()

	// Connect to the VNC server
	vncConn, err := net.Dial("tcp", "localhost:5968")
	if err != nil {
		log.Println("Error connecting to VNC server:", err)
		return
	}
	defer vncConn.Close()

	// Forward messages from the WebSocket to the VNC server
	go func() {
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			_, err = vncConn.Write(message)
			if err != nil {
				log.Println("Write error:", err)
				break
			}
		}
	}()

	// Forward messages from the VNC server to the WebSocket
	buffer := make([]byte, 1024)
	for {
		n, err := vncConn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("Read error:", err)
			}
			break
		}
		err = wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}

}
