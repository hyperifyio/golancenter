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

const (
	// WebSocket readyState values
	WebSocketConnecting = 0 // The connection is not yet open.
	WebSocketOpen       = 1 // The connection is open and ready to communicate.
	WebSocketClosing    = 2 // The connection is in the process of closing.
	WebSocketClosed     = 3 // The connection is closed or couldn't be opened.
)

type Conn struct {
	ctx             context.Context
	ws              js.Value
	wsAddr          string // The address to websocket service like ws://localhost:8080/ws
	network         string // The network protocol for connecting remote service, e.g. "tcp"
	addr            string // The address where to connect, e.g. "1.2.3.4:443"
	readDeadline    time.Time
	writeDeadline   time.Time
	arrayBufferChan chan js.Value // Channel for incoming ArrayBuffer values
	buffer          []byte        // Buffer for storing excess data
	messageHandler  js.Func       // Reference to the message event listener function

	successCallback js.Func
	failureCallback js.Func
}

func (c *Conn) waitReadyState() error {

	// Poll the WebSocket readyState to wait until it's OPEN.
	for {
		log.Printf("conn: reading readyState")
		readyState := c.ws.Get("readyState").Int()
		if readyState == WebSocketOpen {
			log.Printf("conn: readyState: Open")
			break // WebSocket is open, ready to send data
		} else if readyState == WebSocketClosed || readyState == WebSocketClosing {
			log.Printf("conn: readyState: Closed or Closing: %d", readyState)
			return fmt.Errorf("WebSocket is not open (state: %d)", readyState)
		} else {
			log.Printf("conn: readyState: other: %d", readyState)
		}
		// Sleep for a short period before checking again
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (c *Conn) Read(b []byte) (n int, err error) {

	if err := c.waitReadyState(); err != nil {
		return 0, err
	}

	// If there's buffered data, use that first
	if len(c.buffer) > 0 {
		log.Printf("conn: reading from buffer")
		n = copy(b, c.buffer)
		c.buffer = c.buffer[n:] // Remove the data that has been copied out
		log.Printf("conn: Read %d bytes from buffer", n)
		return n, nil
	}

	var deadlineChan <-chan time.Time
	if !c.readDeadline.IsZero() {
		log.Printf("conn: setting up with deadline")
		timer := time.NewTimer(time.Until(c.readDeadline))
		defer timer.Stop() // Ensure the timer is stopped to free resources
		deadlineChan = timer.C
	} else {
		log.Printf("conn: setting up with no deadline")
		// Create a channel that never receives to effectively ignore the deadline
		deadlineChan = make(chan time.Time)
	}

	// Wait for data to arrive or context to be cancelled.
	select {

	case arrayBuffer := <-c.arrayBufferChan:

		log.Printf("conn: Received ArrayBuffer")
		arrayString := stringifyJSObject(arrayBuffer)
		log.Printf("conn: Received ArrayBuffer: %s", arrayString)

		// Convert ArrayBuffer to Uint8Array to work with it in Go
		uint8Array := js.Global().Get("Uint8Array").New(arrayBuffer)

		// Create a Go byte slice with the same length as the Uint8Array
		data := make([]byte, uint8Array.Get("length").Int())

		// Copy the contents of the Uint8Array to the Go byte slice
		js.CopyBytesToGo(data, uint8Array)

		// Send the binary data to the channel
		log.Printf("conn: copying from data channel")
		n = copy(b, data) // Copy as much data as fits into b.
		log.Printf("conn: copied %d bytes", n)
		if n < len(data) {
			log.Printf("conn: Saving to buffer")
			c.buffer = append(c.buffer, data[n:]...)
		}
		return n, nil

	case <-deadlineChan:
		log.Printf("conn: Deadline exit")
		return 0, ErrReadTimeout

	case <-c.ctx.Done():
		log.Printf("conn: Context cancel exit")
		return 0, c.ctx.Err()

	}

}

func (c *Conn) Write(b []byte) (n int, err error) {

	if !c.writeDeadline.IsZero() && !time.Now().Before(c.writeDeadline) {
		log.Printf("conn: write deadline")
		return 0, ErrWriteTimeout
	}

	if err := c.waitReadyState(); err != nil {
		return 0, err
	}

	byteLen := len(b)

	log.Printf("conn: writing %d bytes", byteLen)

	// Convert the Go byte slice to a JavaScript Uint8Array. This is necessary
	// because Go slices are not directly usable as JavaScript typed arrays,
	// but we can use the WebAssembly linear memory to create a view that
	// JavaScript can understand.
	log.Printf("conn: converting to uint8Array of size %d", byteLen)
	uint8Array := js.Global().Get("Uint8Array").New(byteLen)

	bytesCopied := js.CopyBytesToJS(uint8Array, b)
	log.Printf("conn: copied %d bytes", bytesCopied)

	// Send the Uint8Array over the WebSocket. This sends the binary data directly,
	// without converting it to a string, preserving its binary format.
	log.Printf("conn: sending uint8Array")
	c.ws.Call("send", uint8Array)

	// Simplistically assume all data is sent successfully.
	log.Printf("conn: wrote %d bytes", bytesCopied)
	return bytesCopied, nil
}

func (c *Conn) Close() error {
	log.Printf("conn: Close")
	c.successCallback.Release()
	c.failureCallback.Release()
	c.ws.Call("close")
	c.ws.Call("removeEventListener", "message", c.messageHandler)
	c.messageHandler.Release()
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	log.Printf("conn: LocalAddr")
	return &Addr{network: "websocket", address: "client"} // Placeholder implementation.
}

func (c *Conn) RemoteAddr() net.Addr {
	log.Printf("conn: RemoteAddr")
	// This could return the WebSocket server's URL or another meaningful identifier.
	return &Addr{network: "websocket", address: c.addr}
}

func (c *Conn) SetDeadline(t time.Time) error {
	log.Printf("conn: SetDeadline: %s", t)
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	log.Printf("conn: SetReadDeadline: %s", t)
	c.readDeadline = t
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	log.Printf("conn: SetWriteDeadline: %s", t)
	c.writeDeadline = t
	return nil
}

// Initialize to set up the WebSocket connection and event listener
func (c *Conn) Initialize() {

	// Example mechanism for waiting for data to arrive.
	log.Printf("conn: Initializing data channel")
	c.arrayBufferChan = make(chan js.Value, 10)

	c.successCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Printf("conn: Converted Blob to ArrayBuffer: %s", stringifyJSObject(args[0]))
		c.arrayBufferChan <- args[0]
		return nil
	})

	c.failureCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Printf("conn: Failed converting blob: %s", stringifyJSObject(args[0]))
		return nil
	})

	// JavaScript function to handle incoming messages.
	log.Printf("conn: Initializing the message handler function")
	c.messageHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		messageEvent := args[0]
		messageEventString := stringifyJSObject(messageEvent)
		log.Printf("conn: Received message event: %v", messageEventString)

		data := messageEvent.Get("data")

		if data.InstanceOf(js.Global().Get("Blob")) {

			log.Printf("conn: Received Blob %s", data.Type().String())

			// Attach the success and failure callbacks to the promise
			data.Call("arrayBuffer").Call("then", c.successCallback, c.failureCallback).Call("catch", c.failureCallback)

			log.Printf("conn: Started converting Blob %s to ArrayBuffer", data.Type().String())
			return nil
		}

		if data.InstanceOf(js.Global().Get("ArrayBuffer")) {

			c.arrayBufferChan <- data

		} else {
			textDataString := stringifyJSObject(data)
			log.Printf("conn: Received something else %s (and ignoring): %s", data.Type().String(), textDataString)
		}

		return nil
	})

	log.Printf("conn: Setting the message handler")
	c.ws.Call("addEventListener", "message", c.messageHandler)

	log.Printf("conn: OK")

}

// NewConn opens up a client side instance of a remote server connection
// by connecting to a WS proxy server.
//   - ws is the JavaScript websocket instance, e.g. `ws := js.Global().Get("WebSocket").New("ws://localhost:8080/echo")`
func NewConn(
	ctx context.Context,
	wsAddr, network, addr string,
) (*Conn, error) {

	log.Printf("conn: New connection(%s, %s, %s)", wsAddr, network, addr)

	wsURL := fmt.Sprintf("%s?network=%s&address=%s", wsAddr, url.QueryEscape(network), url.QueryEscape(addr))

	log.Printf("conn: wsURL: %s", wsURL)

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

// stringifyJSObject uses JavaScript's JSON.stringify to convert a JS object to a string.
func stringifyJSObject(obj js.Value) string {
	// Get the global JSON object and its stringify method
	JSON := js.Global().Get("JSON")
	stringify := JSON.Get("stringify")

	// Call stringify with the object you want to convert
	result := stringify.Invoke(obj)

	// Convert the result to a Go string and return it
	return result.String()
}
