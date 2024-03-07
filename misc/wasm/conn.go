// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.
//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"syscall/js"
	"time"
)

// Error definitions
var (
	ErrReadTimeout  = errors.New("read timeout exceeded")
	ErrWriteTimeout = errors.New("write timeout exceeded")
)

type Addr struct {
	network string // The network type, e.g., "tcp"
	address string // The server's address, e.g., "192.0.2.1:25"
}

func (a Addr) Network() string {
	return a.network
}

func (a Addr) String() string {
	return a.address
}

var _ net.Addr = (*Addr)(nil)

type Conn struct {
	ctx            context.Context
	ws             js.Value
	wsAddr         string // The address to websocket service like ws://localhost:8080/ws
	network        string // The network protocol for connecting remote service, e.g. "tcp"
	addr           string // The address where to connect, e.g. "1.2.3.4:443"
	readDeadline   time.Time
	writeDeadline  time.Time
	dataChan       chan []byte // Channel for incoming data
	buffer         []byte      // Buffer for storing excess data
	messageHandler js.Func     // Reference to the message event listener function
}

func (c *Conn) Read(b []byte) (n int, err error) {

	// If there's buffered data, use that first
	if len(c.buffer) > 0 {
		n = copy(b, c.buffer)
		c.buffer = c.buffer[n:] // Remove the data that has been copied out
		return n, nil
	}

	var deadlineChan <-chan time.Time
	if !c.readDeadline.IsZero() {
		timer := time.NewTimer(time.Until(c.readDeadline))
		defer timer.Stop() // Ensure the timer is stopped to free resources
		deadlineChan = timer.C
	} else {
		// Create a channel that never receives to effectively ignore the deadline
		deadlineChan = make(chan time.Time)
	}

	// Wait for data to arrive or context to be cancelled.
	select {

	case data := <-c.dataChan:
		n = copy(b, data) // Copy as much data as fits into b.
		if n < len(data) {
			c.buffer = append(c.buffer, data[n:]...)
		}
		return n, nil

	case <-deadlineChan:
		return 0, ErrReadTimeout

	case <-c.ctx.Done():
		return 0, c.ctx.Err()

	}

}

func (c *Conn) Write(b []byte) (n int, err error) {

	if !c.writeDeadline.IsZero() && !time.Now().Before(c.writeDeadline) {
		return 0, ErrWriteTimeout
	}

	// Convert the Go byte slice to a JavaScript Uint8Array. This is necessary
	// because Go slices are not directly usable as JavaScript typed arrays,
	// but we can use the WebAssembly linear memory to create a view that
	// JavaScript can understand.
	uint8Array := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(uint8Array, b)

	// Send the Uint8Array over the WebSocket. This sends the binary data directly,
	// without converting it to a string, preserving its binary format.
	c.ws.Call("send", uint8Array)

	// Simplistically assume all data is sent successfully.
	return len(b), nil
}

func (c *Conn) Close() error {
	c.ws.Call("close")
	c.ws.Call("removeEventListener", "message", c.messageHandler)
	c.messageHandler.Release()
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return &Addr{network: "websocket", address: "client"} // Placeholder implementation.
}

func (c *Conn) RemoteAddr() net.Addr {
	// This could return the WebSocket server's URL or another meaningful identifier.
	return &Addr{network: "websocket", address: c.addr}
}

func (c *Conn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

// Initialize to set up the WebSocket connection and event listener
func (c *Conn) Initialize() {

	// Example mechanism for waiting for data to arrive.
	c.dataChan = make(chan []byte, 10) // Buffer for one message for simplicity.

	// JavaScript function to handle incoming messages.
	c.messageHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		messageEvent := args[0]
		data := messageEvent.Get("data")

		// Check if the incoming message is binary (ArrayBuffer)
		if data.InstanceOf(js.Global().Get("ArrayBuffer")) {

			// Convert ArrayBuffer to Uint8Array to work with it in Go
			uint8Array := js.Global().Get("Uint8Array").New(data)

			// Create a Go byte slice with the same length as the Uint8Array
			goBytes := make([]byte, uint8Array.Get("length").Int())

			// Copy the contents of the Uint8Array to the Go byte slice
			js.CopyBytesToGo(goBytes, uint8Array)

			// Send the binary data to the channel
			c.dataChan <- goBytes

		} else {
			// For text messages, convert to string and then to bytes, as before
			// This branch can be used if you expect mixed content (binary and text)
			textData := data.String()
			c.dataChan <- []byte(textData)
		}

		return nil
	})

	c.ws.Call("addEventListener", "message", c.messageHandler)

}

// NewConn opens up a client side instance of a remote server connection
// by connecting to a WS proxy server.
//   - ws is the JavaScript websocket instance, e.g. `ws := js.Global().Get("WebSocket").New("ws://localhost:8080/echo")`
func NewConn(
	ctx context.Context,
	wsAddr, network, addr string,
) (*Conn, error) {

	log.Printf("New connection(%s, %s, %s)", wsAddr, network, addr)

	wsURL := fmt.Sprintf("%s?network=%s&address=%s", wsAddr, url.QueryEscape(network), url.QueryEscape(addr))

	ws := js.Global().Get("WebSocket").New(wsURL)

	conn := &Conn{
		ctx:     ctx,
		ws:      ws,
		wsAddr:  wsAddr,
		network: network,
		addr:    addr,
	}
	conn.Initialize()
	return conn, nil
}

var _ net.Conn = (*Conn)(nil)
