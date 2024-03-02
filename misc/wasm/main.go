//go:build js && wasm
// +build js,wasm

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"syscall/js"
)

func generateCSR() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	subject := pkix.Name{
		CommonName:         "www.example.com",
		Country:            []string{"US"},
		Province:           []string{""},
		Locality:           []string{""},
		Organization:       []string{"Example Co"},
		OrganizationalUnit: []string{"IT"},
	}

	template := x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	return csrPEM, nil
}

func generateCSRWrapper() func(js.Value, []js.Value) interface{} {
	return func(js.Value, []js.Value) interface{} {
		csr, err := generateCSR()
		if err != nil {
			return err.Error()
		}
		return string(csr)
	}
}

func main() {
	js.Global().Set("generateCSR", js.FuncOf(generateCSRWrapper()))
	<-make(chan bool)
}
