package main

import (
	"log"
	"net/http"
	"github.com/webview/webview_go"
)

func main() {
	// Serve static files from the 'public' directory
	staticFilesDir := "cmd/golancenter/public"
	http.Handle("/", http.FileServer(http.Dir(staticFilesDir)))

	// Handle WebSocket connections for SSH functionality
	http.HandleFunc("/ssh", proxyHandler)

	// Start HTTP server in a goroutine
	go func() {
		log.Println("Starting server on :8080...")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Initialize and configure webview
	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("SSH Client")
	w.SetSize(800, 600, webview.HintNone)
	w.Navigate("http://localhost:8080")
	w.Run()
}
