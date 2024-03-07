// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.
//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

func main() {

	// js.Global().Set("generateCSR", js.FuncOf(generateCSRWrapper()))
	// <-make(chan bool)

	c := make(chan struct{}, 0)

	println("WASM WebSocket Client Starting...")
	// document := js.Global().Get("document")

	tlsConfig := &tls.Config{}

	client := NewClient("ws://localhost:8080/ws", tlsConfig)

	// Use the client as needed...
	resp, err := client.Get("https://www.sendanor.fi/api")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	<-c
}

func NewClient(
	wsAddr string,
	tlsConfig *tls.Config,
) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{

			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return NewConn(ctx, wsAddr, network, addr)
			},

			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
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

// func generateCSRWrapper() func(js.Value, []js.Value) interface{} {
// 	return func(js.Value, []js.Value) interface{} {
// 		csr, err := generateCSR()
// 		if err != nil {
// 			return err.Error()
// 		}
// 		return string(csr)
// 	}
// }
