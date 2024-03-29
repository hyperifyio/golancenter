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

//go:embed novnc/vnc.html novnc/package.json novnc/app/* novnc/core/* novnc/vendor/*
var webContent embed.FS

func main() {

	// Create an http.FileSystem from the embedded files.
	// The "web" subdirectory becomes the root of this file system.
	contentFS, err := fs.Sub(webContent, "novnc")
	if err != nil {
		log.Fatal(err)
	}

	hostname := "localhost"
	port := "8080"
	url := fmt.Sprintf("http://%s:%s", hostname, port)
	listenAddr := fmt.Sprintf("%s:%s", hostname, port)
	vncUrl := fmt.Sprintf("%s/vnc.html?host=%s&port=%s&encrypt=0", url, hostname, port)

	http.HandleFunc("/websockify", proxyHandler)
	http.Handle("/", http.FileServer(http.FS(contentFS)))

	log.Printf("Listening on %s...", url)
	go func() {
		err = http.ListenAndServe(listenAddr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Open a webview window
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("lan.center")
	w.SetSize(800, 600, webview.HintNone)
	w.Navigate(vncUrl)
	w.Run()

}
