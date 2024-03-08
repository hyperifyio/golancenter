// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.
//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"syscall/js"
)

//go:embed certs
var certs embed.FS

func main() {

	c := make(chan struct{}, 0)

	println("wasm: WASM WebSocket Client Starting...")

	rootCaBytes, err := readRootCACert()
	if err != nil {
		panic(err) // or handle the error appropriately
	}

	serverCaBytes, err := readServerCACert()
	if err != nil {
		panic(err) // or handle the error appropriately
	}

	caCertPool, err := loadCACert(rootCaBytes, serverCaBytes)
	if err != nil {
		panic(err) // or handle the error appropriately
	}

	// Read the embedded server certificate and key
	certData, err := certs.ReadFile("certs/client-cert.pem")
	if err != nil {
		log.Fatalf("Failed to read client certificate: %v", err)
	}
	keyData, err := certs.ReadFile("certs/client-key.pem")
	if err != nil {
		log.Fatalf("Failed to read client key: %v", err)
	}

	// Load the client certificate and key
	clientTLS, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		log.Fatalf("Failed to create TLS key pair: %v", err)
	}

	tlsConfig := &tls.Config{
		ServerName:   "localhost",
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientTLS},
	}

	client := NewClient("ws://localhost:8080/ws", tlsConfig)

	// Use the client as needed...
	resp, err := client.Get("https://localhost:8443/")
	if err != nil {
		println(fmt.Sprintf("wasm: Error: %v", err))
		panic(fmt.Errorf("wasm: request failed: %w", err))
	}
	defer resp.Body.Close()

	// Read the response body
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		println("wasm: Error reading response body:", err)
		return
	}

	println(fmt.Sprintf("wasm: Response data: %d", len(responseData)))

	// Convert the response data to a string
	responseString := string(responseData)

	println(fmt.Sprintf("wasm: Response string: %s", responseString))

	// Use syscall/js to print the response to the document
	document := js.Global().Get("document")
	if !document.Truthy() {
		println("wasm: Could not get document object")
		return
	}

	// Create a new <p> element to hold the response
	p := document.Call("createElement", "p")
	p.Set("innerText", responseString)

	// Append the <p> element to the body of the document
	document.Get("body").Call("appendChild", p)

	<-c
}

func NewClient(
	wsAddr string,
	tlsConfig *tls.Config,
) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{

			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				println(fmt.Sprintf("wasm: NewClient: DialContext: %s %s", network, addr))
				return NewConn(ctx, wsAddr, network, addr)
			},

			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				println(fmt.Sprintf("wasm: NewClient: DialTLSContext: %s %s", network, addr))
				conn, err := NewConn(ctx, wsAddr, network, addr)
				if err != nil {
					return nil, err
				}
				tlsConn := tls.Client(conn, tlsConfig)
				return tlsConn, nil
			},
		},
	}
	return client
}

func readRootCACert() ([]byte, error) {
	return certs.ReadFile("certs/root-ca-cert.pem")
}

func readServerCACert() ([]byte, error) {
	return certs.ReadFile("certs/server-ca-cert.pem")
}

func loadCACert(
	rootCa []byte,
	serverCa []byte,
) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(rootCa) {
		return nil, fmt.Errorf("failed to append root CA certificate")
	}
	if !caCertPool.AppendCertsFromPEM(serverCa) {
		return nil, fmt.Errorf("failed to append server CA certificate")
	}
	return caCertPool, nil
}
