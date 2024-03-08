// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/webview/webview_go"
)

//go:embed web/*
var webContent embed.FS

func main() {

	// Local HTTP server acting as a proxy for outside world connections
	//
	// These websocket connections will be encrypted by TLS, so HTTP is suitable,
	// although in a production version we should verify only verified users
	// can access it. (Encrypting it though is not necessary, since it would be
	// double encrypted then and bad for performance.) Signed JWT might do the
	// trick.
	hostname := "localhost"
	port := "8080"
	listenAddr := fmt.Sprintf("%s:%s", hostname, port)

	contentFS, err := fs.Sub(webContent, "web")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/ws", netConnHandler())
	http.Handle("/", http.FileServer(http.FS(contentFS)))

	log.Printf("Listening on %s...", listenAddr)
	go func() {
		err = http.ListenAndServe(listenAddr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Open a webview window which starts a mTLS request to outside world
	listenUrl := fmt.Sprintf("http://%s:%s", hostname, port)
	viewUrl := fmt.Sprintf("%s/index.html", listenUrl)
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("lan.center")
	w.SetSize(800, 600, webview.HintNone)
	w.Navigate(viewUrl)
	w.Run()

}
