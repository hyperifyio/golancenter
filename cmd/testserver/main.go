// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {

	// Handler function returns "Hello World"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Hello World")
		if err != nil {
			log.Printf("formating failed: %v", err)
		}
	})

	caCertPool := x509.NewCertPool()

	// Load CA certificate
	caCertPath := "./certs/root-ca-cert.pem" // Path to your CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		log.Fatalf("Failed to append CA certificate to pool")
	}

	// Load intermediate client CA certificate
	caClientCertPath := "./certs/client-ca-cert.pem" // Path to your CA certificate
	caClientCert, err := os.ReadFile(caClientCertPath)
	if err != nil {
		log.Fatalf("Failed to read intermediate client CA certificate: %v", err)
	}
	ok = caCertPool.AppendCertsFromPEM(caClientCert)
	if !ok {
		log.Fatalf("Failed to append intermadiate client CA certificate to pool")
	}

	// Read the embedded server certificate and key
	certData, err := os.ReadFile("certs/server-cert.pem")
	if err != nil {
		log.Fatalf("Failed to read server certificate: %v", err)
	}
	keyData, err := os.ReadFile("certs/server-key.pem")
	if err != nil {
		log.Fatalf("Failed to read server key: %v", err)
	}

	// Load the server certificate and key
	serverTLS, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		log.Fatalf("Failed to create TLS key pair: %v", err)
	}

	// Create a TLS configuration with the CA certificate pool
	tlsConfig := &tls.Config{
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Require and verify client certificates
		Certificates: []tls.Certificate{serverTLS},
	}

	// Create a custom server with TLS configuration
	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	// Paths to your server's certificate and key files
	certPath := "./certs/server-cert.pem"
	keyPath := "./certs/server-key.pem"

	// Start the HTTPS server with TLS configuration
	log.Printf("Starting HTTPS server on https://localhost:8443")
	err = server.ListenAndServeTLS(certPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}

}
